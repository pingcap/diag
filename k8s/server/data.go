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
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pingcap/diag/api/types"
	"github.com/pingcap/diag/pkg/utils"
)

// getDataList implements GET /data
func getDataList(c *gin.Context) {
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

	status := strings.ToLower(c.Param("status"))

	dataList := make([]types.DataSet, 0)
	for _, job := range diagCtx.getCollectJobs() {
		switch status {
		case "":
			if job.Status != taskStatusFinish {
				continue // only list finished jobs by default
			}
		case "all":
			// do nothing, accept all jobs
		case taskStatusAccepted,
			taskStatusCancel,
			taskStatusError,
			taskStatusPurge,
			taskStatusFinish,
			taskStatusRunning:
			if job.Status != status {
				continue
			}
		default:
			sendErrMsg(c, http.StatusBadRequest, "unknown status")
			return
		}

		data, err := buildDataFromJob(job)
		if err != nil {
			sendErrMsg(c, http.StatusInternalServerError, err.Error())
			return // maybe log error and continue?
		}
		dataList = append(dataList, data)
	}

	c.JSON(http.StatusOK, dataList)
}

// getDataSet implements GET /data/{id}
func getDataSet(c *gin.Context) {
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
	if job == nil || job.Status == taskStatusPurge {
		msg := fmt.Sprintf("data set for collect job '%s' not found", id)
		sendErrMsg(c, http.StatusNotFound, msg)
		return
	}
	data, err := buildDataFromJob(job)
	if err != nil {
		sendErrMsg(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, data)
}

// deleteDataSet implements DELETE /data/{id}
func deleteDataSet(c *gin.Context) {
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
	if job == nil || job.Status == taskStatusPurge {
		msg := fmt.Sprintf("data set for collect job '%s' not found", id)
		sendErrMsg(c, http.StatusNotFound, msg)
		return
	}
	if job.Status == taskStatusAccepted ||
		job.Status == taskStatusRunning ||
		job.Dir == "" {
		msg := fmt.Sprintf("collect job '%s' not finished yet", id)
		sendErrMsg(c, http.StatusServiceUnavailable, msg)
		return
	}

	diagCtx.Lock()
	defer diagCtx.Unlock()
	if err := os.RemoveAll(job.Dir); err != nil {
		msg := fmt.Sprintf("failed removing data set '%s': %s", id, err)
		sendErrMsg(c, http.StatusInternalServerError, msg)
		return
	}
	job.Status = taskStatusPurge

	c.JSON(http.StatusNoContent, nil)
}

// buildDataFromJob creates a new DataSet object with information from
// the matched CollectJob
func buildDataFromJob(job *types.CollectJob) (types.DataSet, error) {
	data := types.DataSet{
		ID:          job.ID,
		Date:        job.Date,
		ClusterName: job.ClusterName,
	}
	dir := job.Dir

	// calculate total file size of the data set
	size, err := utils.DirSize(dir)
	if err != nil && !os.IsNotExist(err) {
		return data, fmt.Errorf("failed to read data set: %s", err)
	}
	data.Size = size

	return data, nil
}
