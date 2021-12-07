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
	goctx "context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/api/types"
	"github.com/pingcap/diag/checker"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"k8s.io/klog/v2"
)

// checkDataSet implements POST /data/{id}/check
func checkDataSet(c *gin.Context) {
	id := c.Param("id")

	// parse argument from POST body
	var req types.CheckDataRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		msg := fmt.Sprintf("invalid request: %s", err)
		sendErrMsg(c, http.StatusBadRequest, msg)
		return
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

	worker := diagCtx.getCollectWorker(id)
	if worker == nil || worker.job == nil ||
		worker.job.Dir == "" || worker.job.Status == taskStatusPurge {
		msg := fmt.Sprintf("data set for collect job '%s' not found", id)
		sendErrMsg(c, http.StatusNotFound, msg)
		return
	}

	// run checker
	runChecker(diagCtx, worker, &req)

	// read output
	output, err := getCheckerOutputString(diagCtx, id)
	if err != nil {
		sendErrMsg(c, http.StatusNotFound, err.Error())
		return
	}

	c.String(http.StatusOK, output)
}

func runChecker(
	ctx *context,
	worker *collectJobWorker,
	req *types.CheckDataRequest,
) {
	// populate logger for the collect job
	cLogger := logprinter.NewLogger("")
	cLogger.SetDisplayMode(logprinter.DisplayModePlain)
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	cLogger.SetStdout(outW)
	cLogger.SetStderr(errW)

	// reset output buffers as we only store the latest checker result
	ctx.Lock()
	worker.checker.reset()
	ctx.Unlock()

	// pipe the outputs
	go func() {
		s := bufio.NewScanner(outR)
		for s.Scan() {
			worker.checker.stdout = append(worker.checker.stdout, s.Bytes()...)
			worker.checker.stdout = append(worker.checker.stdout, '\n')
		}
		if err := s.Err(); err != nil {
			klog.Error("error getting stdout of collect job %s: %s", worker.job.ID, err)
		}
	}()
	go func() {
		s := bufio.NewScanner(errR)
		for s.Scan() {
			worker.checker.stdout = append(worker.checker.stderr, s.Bytes()...)
			worker.checker.stdout = append(worker.checker.stdout, '\n')
		}
		if err := s.Err(); err != nil {
			klog.Error("error getting stdout of collect job %s: %s", worker.job.ID, err)
		}
	}()

	opt := checker.NewOptions()
	opt.Inc = req.Types
	opt.DataPath = worker.job.Dir

	doneChan := make(chan struct{}, 1)
	errChan := make(chan error, 1)
	go func() {
		err := opt.RunChecker(
			goctx.WithValue(
				goctx.Background(),
				logprinter.ContextKeyLogger,
				cLogger,
			),
		)
		outW.Close()
		errW.Close()
		if err != nil {
			errChan <- err
			return
		}
		doneChan <- struct{}{}
	}()

	select {
	case <-worker.checker.cancel:
		klog.Infof("check for collect job %s cancelled.", worker.job.ID)
	case err := <-errChan:
		klog.Errorf("check for collect job %s failed with error: %s", worker.job.ID, err)
	case <-doneChan:
		klog.Infof("check for collect job %s finished.", worker.job.ID)
	}

	// mark the checker as done no matter what result it is to indicate its result
	// is ready to read
	ctx.Lock()
	defer ctx.Unlock()
	worker.checker.finished = true
}

// getCheckResult implements GET /data/{id}/check
func getCheckResult(c *gin.Context) {
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

	output, err := getCheckerOutputString(diagCtx, id)
	if err != nil {
		sendErrMsg(c, http.StatusNotFound, err.Error())
		return
	}

	c.String(http.StatusOK, output)
}

// cancelCheck implements DELETE /data/{id}/check
func cancelCheck(c *gin.Context) {
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
	if worker.checker == nil {
		sendErrMsg(c, http.StatusNotFound,
			fmt.Sprintf("checker for collect job '%s' is not available", id))
		return
	}

	diagCtx.Lock()
	defer diagCtx.Unlock()

	if worker.checker.finished {
		output, err := getCheckerOutputString(diagCtx, id)
		if err != nil {
			sendErrMsg(c, http.StatusNotFound, err.Error())
			return
		}

		c.String(http.StatusAccepted, output)
	}

	worker.checker.cancel <- struct{}{}

	c.JSON(http.StatusNoContent, nil)
}

func getCheckerOutputString(ctx *context, id string) (string, error) {
	stdout, stderr, ok := ctx.getCheckOutputs(id)
	if !ok {
		return "", fmt.Errorf("collect job '%s' does not exist", id)
	}

	var output string
	if stdout != nil {
		output = fmt.Sprintf("stdout:\n%s\n", stdout)
	}
	if stderr != nil {
		output = fmt.Sprintf("%sstderr:\n%s\n", output, stderr)
	}
	return output, nil
}
