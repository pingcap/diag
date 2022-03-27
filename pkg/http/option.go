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

type HTTPOption func(task *HTTPCollectTask)

// WithTimeOut  set http request timeout
func WithTimeOut(timeout time.Duration) HTTPOption {
	return func(task *HTTPCollectTask) {
		if timeout <= 0 {
			task.timeout = 10 * time.Second * 10
			return
		}
		task.timeout = timeout
	}
}

// WithHeader set http request head
func WithHeader(header map[string]string) HTTPOption {
	return func(task *HTTPCollectTask) {
		task.header = header
	}
}

// WithTLSCfg  set http request tls config
func WithTLSCfg(tlsCfg *tls.Config) HTTPOption {
	return func(task *HTTPCollectTask) {
		task.tlsCfg = tlsCfg
	}
}

// WithMethod set http request method
func WithMethod(method string) HTTPOption {
	return func(task *HTTPCollectTask) {
		switch strings.ToUpper(method) {
		case "GET":
			task.method = method
		case "POST":
			task.method = method
		default:
			task.method = "GET"
		}
	}
}
