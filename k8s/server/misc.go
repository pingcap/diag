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
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/pingcap/diag/api/types"
	"github.com/pingcap/diag/version"
	"k8s.io/klog/v2"
)

// getVersion implements GET /version
func getVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": version.String(),
		"go":      runtime.Version(),
	})
}

// sendErrMsg responses error message to client and log it
func sendErrMsg(c *gin.Context, status int, msg string) {
	klog.Errorf(msg)
	c.JSON(status, types.ResponseMsg{
		Message: msg,
	})
}

// getStatus implements GET /status
func getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, struct{}{})
}
