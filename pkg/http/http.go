// Copyright 2022 PingCAP, Inc.
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

package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pingcap/tiup/pkg/utils"
)

// HTTPCollectTask  collect data via http request
type HTTPCollectTask struct {
	filePath string
	url      string
	header   map[string]string
	timeout  time.Duration
	tlsCfg   *tls.Config
	method   string
}

// NewHTTPTask
func NewHTTPTask(filePath, url string, opts ...HTTPOption) *HTTPCollectTask {
	job := &HTTPCollectTask{
		filePath: filePath,
		url:      url,
		method:   "GET", // default method is GET
		tlsCfg:   nil,
		timeout:  10 * time.Second,
	}

	// withOptions
	for _, opt := range opts {
		opt(job)
	}
	return job
}

// Do  sent http request to url and save response
func (task *HTTPCollectTask) Do(ctx context.Context) error {

	scheme := "http"
	if task.tlsCfg != nil {
		scheme = "https"
	}

	// new http client
	httpClient := utils.NewHTTPClient(task.timeout, task.tlsCfg)

	if task.header != nil {
		for k, v := range task.header {
			httpClient.SetRequestHeader(k, v)
		}
	}

	url := fmt.Sprintf("%s://%s", scheme, task.url)
	if err := utils.CreateDir(filepath.Dir(task.filePath)); err != nil {
		return err
	}

	switch strings.ToUpper(task.method) {
	case "GET":
		err := httpClient.Download(ctx, url, task.filePath)
		if err != nil {
			return err
		}
	case "POST":
		return nil

	default:
		return fmt.Errorf("unknown http request method %s", task.method)
	}

	return nil
}
