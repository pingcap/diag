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

// HttpCollectJob  collect data via http request
type HttpCollectJob struct {
	filePath string
	url      string
	header   map[string]string
	timeout  time.Duration
	tlsCfg   *tls.Config
	method   string
}

// NewHttpJob
func NewHttpJob(filePath, url string, opts ...httpOption) *HttpCollectJob {
	job := &HttpCollectJob{
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
func (job *HttpCollectJob) Do(ctx context.Context) error {

	scheme := "http"
	if job.tlsCfg != nil {
		scheme = "https"
	}

	// new http client
	httpClient := utils.NewHTTPClient(job.timeout, job.tlsCfg)

	if job.header != nil {
		for k, v := range job.header {
			httpClient.SetRequestHeader(k, v)
		}
	}

	url := fmt.Sprintf("%s://%s", scheme, job.url)
	if err := utils.CreateDir(filepath.Dir(job.filePath)); err != nil {
		return err
	}

	if strings.ToUpper(job.method) == "GET" {
		err := httpClient.Download(ctx, url, job.filePath)
		if err != nil {
			return err
		}
	}

	return nil
}
