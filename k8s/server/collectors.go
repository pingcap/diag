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
	"fmt"
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
	collectJobStatusAccepted = "accepted"
	collectJobStatusRunning  = "running"
	collectJobStatusError    = "error"
	collectJobStatusFinish   = "finished"
	collectJobStatusCancel   = "cancelled"
	collectJobStatusPurge    = "purged" // data set deleted
)

// collectData implements POST /collectors
func collectData(c *gin.Context) {
	currTime := time.Now()

	// TODO: parse argument from POST body
	var req types.CollectJobRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		msg := fmt.Sprintf("invalid request: %s", err)
		sendErrMsg(c, http.StatusBadRequest, msg)
		return
	}

	// build collect job
	opt := collector.BaseOptions{
		Cluster:     req.ClusterName,
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
		Status:      collectJobStatusAccepted,
		ClusterName: opt.Cluster,
		Collectors:  collectors,
		From:        opt.ScrapeBegin,
		To:          opt.ScrapeEnd,
		Date:        currTime.Format(time.RFC3339),
	}
	worker := diagCtx.insertCollectJob(job)

	// run collector
	go runCollectors(diagCtx, &opt, worker)

	c.JSON(http.StatusAccepted, worker.job)
}

func runCollectors(
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

	cLogger := logprinter.NewLogger("")
	cLogger.SetDisplayMode(logprinter.DisplayModePlain)
	cm := collector.NewEmptyManager("tidb", worker.job.ID, cLogger)

	doneChan := make(chan struct{}, 1)
	errChan := make(chan error, 1)
	go func() {
		ctx.setJobStatus(worker.job.ID, collectJobStatusRunning)
		resultDir, err := cm.CollectClusterInfo(opt, &cOpt, &gOpt, ctx.kubeCli, ctx.dynCli)
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
		klog.Infof("collect job %s cancelled.", worker.job.ID)
		ctx.setJobStatus(worker.job.ID, collectJobStatusCancel)
	case err := <-errChan:
		klog.Errorf("collect job %s failed with error: %s", worker.job.ID, err)
		ctx.setJobStatus(worker.job.ID, collectJobStatusError)
	case <-doneChan:
		klog.Infof("collect job %s finished.", worker.job.ID)
		ctx.setJobStatus(worker.job.ID, collectJobStatusFinish)
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
