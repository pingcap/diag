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
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/set"
	klog "k8s.io/klog"
)

// collect job status
const (
	collectJobStatusAccepted = "accepted"
	collectJobStatusRunning  = "running"
	collectJobStatusError    = "error"
	collectJobStatusFinish   = "finished"
	collectJobStatusCancel   = "cancelled"
)

// collectData implements POST /collectors
func collectData(c *gin.Context) {
	currTime := time.Now()

	// TODO: parse argument from POST body
	var req types.CollectJobRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		msg := fmt.Sprintf("invalid request: %s", err)
		returnErrMsg(c, http.StatusBadRequest, msg)
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

	ctx, ok := c.Get(diagAPICtxKey)
	if !ok {
		msg := "failed to read server config."
		returnErrMsg(c, http.StatusInternalServerError, msg)
		return
	}
	diagCtx, ok := ctx.(*context)
	if !ok {
		msg := "server config is in wrong type."
		returnErrMsg(c, http.StatusInternalServerError, msg)
		return
	}

	job := &types.CollectJob{
		Status:      collectJobStatusAccepted,
		ClusterName: opt.Cluster,
		From:        opt.ScrapeBegin,
		To:          opt.ScrapeEnd,
		Date:        currTime.Format(time.RFC3339),
	}
	jobInfo := diagCtx.insertCollectJob(job)

	// run collector
	go runCollectors(diagCtx, &opt, jobInfo)

	c.JSON(http.StatusAccepted, types.CollectJob{})
}

func runCollectors(
	ctx *context,
	opt *collector.BaseOptions,
	jobInfo *collectJobCtx,
) {
	gOpt := operator.Options{Concurrency: 2}
	cOpt := collector.CollectOptions{
		Include: set.NewStringSet( // collect all available types by default
			collector.CollectTypeMonitor,
			collector.CollectTypeConfig,
		),
		Exclude: set.NewStringSet(),
		Mode:    collector.CollectModeK8s,
	}
	cm := collector.NewEmptyManager("tidb")

	done := make(chan struct{})
	go func() {
		err := cm.CollectClusterInfo(opt, &cOpt, &gOpt, ctx.kubeCli, ctx.dynCli)
		if err != nil {
			klog.Errorf("error collecting info: %s", err)
			ctx.Lock()
			jobInfo.job.Status = collectJobStatusError
			ctx.Unlock()
		}
		ctx.Lock()
		jobInfo.job.Status = collectJobStatusFinish
		ctx.Unlock()
		done <- struct{}{}
	}()

	select {
	case <-jobInfo.cancel:
		klog.Infof("collect job %s cancelled.", jobInfo.job.ID)
		ctx.Lock()
		jobInfo.job.Status = collectJobStatusCancel
		ctx.Unlock()
	case <-done:
		klog.Infof("collect job %s finished.", jobInfo.job.ID)
	}
}
