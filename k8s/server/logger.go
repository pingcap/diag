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
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
)

// ginLogger creates a logger middleware for gin with klog
func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// necessary variables
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// process request
		c.Next()

		// handle logging
		end := time.Now()
		latency := end.Sub(start)

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				klog.Error(e)
			}
		}
		klog.Infof("request handled: %d %s %s/%s, %s, %s",
			c.Writer.Status(), c.Request.Method, path, query,
			c.Request.UserAgent(), latency,
		)
	}
}
