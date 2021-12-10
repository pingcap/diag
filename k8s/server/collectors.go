// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/api/types"
	"github.com/pingcap/diag/collector"
	"github.com/pingcap/tiup/pkg/base52"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/crypto/rand"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/set"
	"k8s.io/klog/v2"
)

// collect job status
const (
	taskStatusAccepted = "accepted"
	taskStatusRunning  = "running"
	taskStatusError    = "error"
	taskStatusFinish   = "finished"
	taskStatusCancel   = "cancelled"
	taskStatusPurge    = "purged" // data set deleted
)

// collectData implements POST /collectors
func collectData(c *gin.Context) {
	currTime := time.Now()

	// parse argument from POST body
	var req types.CollectJobRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		msg := fmt.Sprintf("invalid request: %s", err)
		sendErrMsg(c, http.StatusBadRequest, msg)
		return
	}

	// build collect job
	opt := collector.BaseOptions{
		Cluster:     req.ClusterName,
		Namespace:   req.Namespace,
		ScrapeBegin: req.From,
		ScrapeEnd:   req.To,
	}

	// parsing time
	if opt.ScrapeBegin == "" {
		opt.ScrapeBegin = time.Now().Add(time.Hour * -2).Format(time.RFC3339)
	}
	if opt.ScrapeEnd == "" {
		opt.ScrapeEnd = time.Now().Format(time.RFC3339)
	}

	// parsing collector list
	collectors := req.Collectors
	if len(req.Collectors) < 1 {
		collectors = []string{
			collector.CollectTypeConfig,
			collector.CollectTypeMonitor,
		}
	}

	ctx, ok := c.Get(diagAPICtxKey)
	if !ok {
		msg := "failed to read server config."
		sendErrMsg(c, http.StatusInternalServerError, msg)
		return
	}
	diagCtx, ok := ctx.(*context)
	if !ok {
		msg := "server config is in wrong type."
		sendErrMsg(c, http.StatusInternalServerError, msg)
		return
	}

	job := &types.CollectJob{
		ID:          base52.Encode(currTime.UnixNano() + rand.Int63n(1000)),
		Status:      taskStatusAccepted,
		ClusterName: fmt.Sprintf("%s/%s", opt.Namespace, opt.Cluster),
		Collectors:  collectors,
		From:        opt.ScrapeBegin,
		To:          opt.ScrapeEnd,
		Date:        currTime.Format(time.RFC3339),
	}
	worker := diagCtx.insertCollectJob(job)

	// run collector
	go runCollector(diagCtx, &opt, worker)

	c.JSON(http.StatusAccepted, worker.job)
}

func runCollector(
	ctx *context,
	opt *collector.BaseOptions,
	worker *collectJobWorker,
) {
	gOpt := operator.Options{Concurrency: 2}
	cOpt := collector.CollectOptions{
		Include: set.NewStringSet(worker.job.Collectors...),
		Exclude: set.NewStringSet(),
		Mode:    collector.CollectModeK8s,
	}

	// populate logger for the collect job
	cLogger := logprinter.NewLogger("")
	cLogger.SetDisplayMode(logprinter.DisplayModePlain)
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	cLogger.SetStdout(outW)
	cLogger.SetStderr(errW)

	// pipe the outputs
	go func() {
		s := bufio.NewScanner(outR)
		for s.Scan() {
			worker.stdout = append(worker.stdout, s.Bytes()...)
			worker.stdout = append(worker.stdout, '\n')
		}
		if err := s.Err(); err != nil {
			klog.Error("error getting stdout of collect job %s: %s", worker.job.ID, err)
		}
	}()
	go func() {
		s := bufio.NewScanner(errR)
		for s.Scan() {
			worker.stdout = append(worker.stderr, s.Bytes()...)
			worker.stdout = append(worker.stdout, '\n')
		}
		if err := s.Err(); err != nil {
			klog.Error("error getting stdout of collect job %s: %s", worker.job.ID, err)
		}
	}()

	cm := collector.NewEmptyManager("tidb", worker.job.ID, cLogger)

	doneChan := make(chan struct{}, 1)
	errChan := make(chan error, 1)
	go func() {
		ctx.setJobStatus(worker.job.ID, taskStatusRunning)
		resultDir, err := cm.CollectClusterInfo(opt, &cOpt, &gOpt, ctx.kubeCli, ctx.dynCli)
		outW.Close()
		errW.Close()
		if err != nil {
			errChan <- err
			return
		}

		ctx.Lock()
		defer ctx.Unlock()

		worker.job.Dir = resultDir
		doneChan <- struct{}{}
	}()

	select {
	case <-worker.cancel:
		// status is updated in the cancel handling function
		klog.Infof("collect job %s cancelled.", worker.job.ID)
	case err := <-errChan:
		klog.Errorf("collect job %s failed with error: %s", worker.job.ID, err)
		ctx.setJobStatus(worker.job.ID, taskStatusError)
	case <-doneChan:
		klog.Infof("collect job %s finished.", worker.job.ID)
		ctx.setJobStatus(worker.job.ID, taskStatusFinish)
	}
}

// collectData implements GET /collectors
func getJobList(c *gin.Context) {
	ctx, ok := c.Get(diagAPICtxKey)
	if !ok {
		msg := "failed to read server config."
		sendErrMsg(c, http.StatusInternalServerError, msg)
		return
	}
	diagCtx, ok := ctx.(*context)
	if !ok {
		msg := "server config is in wrong type."
		sendErrMsg(c, http.StatusInternalServerError, msg)
		return
	}

	c.JSON(http.StatusOK, diagCtx.getCollectJobs())
}

// getCollectJob implements GET /collectors/{id}
func getCollectJob(c *gin.Context) {
	id := c.Param("id")

	ctx, ok := c.Get(diagAPICtxKey)
	if !ok {
		msg := "failed to read server config."
		sendErrMsg(c, http.StatusInternalServerError, msg)
		return
	}
	diagCtx, ok := ctx.(*context)
	if !ok {
		msg := "server config is in wrong type."
		sendErrMsg(c, http.StatusInternalServerError, msg)
		return
	}

	job := diagCtx.getCollectJob(id)
	if job == nil {
		sendErrMsg(c, http.StatusNotFound,
			fmt.Sprintf("collect job '%s' does not exist", id))
		return
	}

	c.JSON(http.StatusOK, job)
}

// cancelCollectJob implements DELETE /collectors/{id}
func cancelCollectJob(c *gin.Context) {
	id := c.Param("id")

	ctx, ok := c.Get(diagAPICtxKey)
	if !ok {
		msg := "failed to read server config."
		sendErrMsg(c, http.StatusInternalServerError, msg)
		return
	}
	diagCtx, ok := ctx.(*context)
	if !ok {
		msg := "server config is in wrong type."
		sendErrMsg(c, http.StatusInternalServerError, msg)
		return
	}

	worker := diagCtx.getCollectWorker(id)
	if worker == nil || worker.job == nil {
		sendErrMsg(c, http.StatusNotFound,
			fmt.Sprintf("collect job '%s' does not exist", id))
		return
	}

	diagCtx.Lock()
	defer diagCtx.Unlock()

	if worker.job.Status == taskStatusCancel {
		c.JSON(http.StatusGone, worker.job)
		return
	}

	worker.job.Status = taskStatusCancel
	worker.cancel <- struct{}{}

	c.JSON(http.StatusAccepted, worker.job)
}

// getCollectLogs implements GET /collectors/{id}/logs
func getCollectLogs(c *gin.Context) {
	id := c.Param("id")

	ctx, ok := c.Get(diagAPICtxKey)
	if !ok {
		msg := "failed to read server config."
		sendErrMsg(c, http.StatusInternalServerError, msg)
		return
	}
	diagCtx, ok := ctx.(*context)
	if !ok {
		msg := "server config is in wrong type."
		sendErrMsg(c, http.StatusInternalServerError, msg)
		return
	}

	stdout, stderr, ok := diagCtx.getCollectJobOutputs(id)
	if !ok {
		sendErrMsg(c, http.StatusNotFound,
			fmt.Sprintf("collect job '%s' does not exist", id))
		return
	}

	var output string
	if stdout != nil {
		output = fmt.Sprintf("stdout:\n%s\n", stdout)
	}
	if stderr != nil {
		output = fmt.Sprintf("%sstderr:\n%s\n", output, stderr)
	}

	c.String(http.StatusOK, output)
}
