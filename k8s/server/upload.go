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
	goctx "context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pingcap/diag/api/types"
	"github.com/pingcap/diag/pkg/packager"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"k8s.io/klog/v2"
)

// uploadDataSet implements POST /data/{id}/upload
func uploadDataSet(c *gin.Context) {
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
	if worker == nil || worker.job == nil ||
		worker.job.Dir == "" || worker.job.Status == taskStatusPurge {
		msg := fmt.Sprintf("data set for collect job '%s' not found", id)
		sendErrMsg(c, http.StatusNotFound, msg)
		return
	}

	// run uploader
	go runUploader(diagCtx, worker)

	c.JSON(http.StatusAccepted, nil)
}

func runUploader(
	ctx *context,
	worker *collectJobWorker,
) {
	// get credentials from environment variables
	// this need to be changed to use proper client authentication method
	// once the clinic server implemented so.
	clinicUsername := os.Getenv("CLINIC_USERNAME")
	clinicPassword := os.Getenv("CLINIC_PASSWORD")
	if clinicUsername == "" || clinicPassword == "" {
		klog.Error("failed to get CLINIC_USERNAME and CLINIC_PASSWORD env vars.")
		ctx.Lock()
		defer ctx.Unlock()
		worker.uploader.status = taskStatusError
		worker.uploader.result = "no credentials available"
		return
	}

	// populate logger for the collect job
	cLogger := logprinter.NewLogger("")
	cLogger.SetDisplayMode(logprinter.DisplayModePlain)
	_, outW := io.Pipe()
	_, errW := io.Pipe()
	cLogger.SetStdout(outW)
	cLogger.SetStderr(errW)

	// reset output buffers as we only store the latest upload result
	ctx.Lock()
	worker.uploader.reset()
	ctx.Unlock()

	doneChan := make(chan string, 1)
	errChan := make(chan error, 1)
	go func() {
		ctx.Lock()
		worker.uploader.status = taskStatusRunning
		ctx.Unlock()

		// package the data set
		pOpt := &packager.PackageOptions{
			InputDir: worker.job.Dir,
			CertPath: "/var/lib/clinic-cert/pingcap.crt", // mounted via secret
		}
		pf, err := packager.PackageCollectedData(pOpt)
		outW.Close()
		errW.Close()
		if err != nil {
			errChan <- err
			return
		}
		klog.Infof("data set of collect job %s packaged as %s", worker.job.ID, pf)

		// upload the packaged data set
		endpoint := "https://clinic.pingcap.com:4433"
		uOpt := &packager.UploadOptions{
			FilePath: pf,
			ClientOptions: packager.ClientOptions{
				Endpoint: endpoint,
				UserName: clinicUsername,
				Password: clinicPassword,
				Client:   packager.InitClient(endpoint),
			},
		}
		ctx := goctx.WithValue(
			goctx.Background(),
			logprinter.ContextKeyLogger,
			cLogger,
		)
		result, err := packager.Upload(ctx, uOpt)
		if err != nil {
			errChan <- err
			return
		}
		klog.Infof("data set of collect job %s uploaded to %s", worker.job.ID, endpoint)

		doneChan <- result
	}()

	select {
	case <-worker.uploader.cancel:
		klog.Infof("uploading for data set of collect job %s cancelled.", worker.job.ID)
		ctx.Lock()
		defer ctx.Unlock()
		worker.uploader.status = taskStatusCancel
	case err := <-errChan:
		klog.Errorf("uploading for data set of collect job %s failed with error: %s", worker.job.ID, err)
		ctx.Lock()
		defer ctx.Unlock()
		worker.uploader.status = taskStatusError
		worker.uploader.result = fmt.Sprintf("error packaging data set: %s", err)
	case result := <-doneChan:
		klog.Infof("uploading for data set of collect job %s finished.", worker.job.ID)
		ctx.Lock()
		defer ctx.Unlock()
		worker.uploader.status = taskStatusFinish
		worker.uploader.result = result
	}
}

// cancelDataUpload implements DELETE /data/{id}/upload
func cancelDataUpload(c *gin.Context) {
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
	if worker.uploader == nil {
		sendErrMsg(c, http.StatusNotFound,
			fmt.Sprintf("checker for collect job '%s' is not available", id))
		return
	}

	diagCtx.Lock()
	defer diagCtx.Unlock()

	if worker.uploader.status == taskStatusFinish {
		task, err := buildUploadTask(diagCtx, id)
		if err != nil {
			sendErrMsg(c, http.StatusAccepted, err.Error())
			return
		}

		c.JSON(http.StatusAccepted, task)
	}

	worker.uploader.cancel <- struct{}{}

	c.JSON(http.StatusNoContent, nil)
}

// getUploadTask implements GET /data/{id}/upload
func getUploadTask(c *gin.Context) {
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

	task, err := buildUploadTask(diagCtx, id)
	if err != nil {
		sendErrMsg(c, http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, task)
}

func buildUploadTask(ctx *context, id string) (*types.UploadTask, error) {
	date, status, result, ok := ctx.getUploadOutputs(id)
	if !ok {
		return nil, fmt.Errorf("collect job '%s' does not exist", id)
	}

	if status == "" {
		return nil, fmt.Errorf("no upload task available for data set %s", id)
	}

	return &types.UploadTask{
		ID:     id,
		Date:   date.Format(time.RFC3339),
		Status: status,
		Result: result,
	}, nil
}
