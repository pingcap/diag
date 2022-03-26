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
	"crypto/tls"
	"strings"
	"time"
)

type httpOption func(job *HttpCollectJob)

// WithTimeOut  set http request timeout
func WithTimeOut(timeout time.Duration) httpOption {
	return func(job *HttpCollectJob) {
		if timeout <= 0 {
			job.timeout = 10 * time.Second * 10
			return
		}
		job.timeout = timeout
	}
}

// WithHeader set http request head
func WithHeader(header map[string]string) httpOption {
	return func(job *HttpCollectJob) {
		job.header = header
	}
}

// WithTlsCfg set http request tls config
func WithTlsCfg(tlsCfg *tls.Config) httpOption {
	return func(job *HttpCollectJob) {
		job.tlsCfg = tlsCfg
	}
}

// WithMethod set http request method
func WithMethod(method string) httpOption {
	return func(job *HttpCollectJob) {
		switch strings.ToUpper(method) {
		case "GET":
			job.method = method
		case "POST":
			job.method = method
		default:
			job.method = "GET"
		}
	}
}
