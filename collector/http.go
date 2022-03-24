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

package collector

import (
	"context"
	"crypto/tls"
	"fmt"
	"path/filepath"
	"time"

	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/utils"
)

// httpRequest  Get data via http request
type httpRequest struct {
	fileName string //
	fileDir  string
	url      string
	header   map[string]string
	timeout  time.Duration
	tlsCfg   *tls.Config
}

// newHTTPRequest
func newHTTPRequest(file, dir, url string, timeout time.Duration, tlsCfg *tls.Config, header map[string]string) httpRequest {
	return httpRequest{
		fileName: file,
		fileDir:  dir,
		url:      url,
		timeout:  timeout,
		header:   header,
		tlsCfg:   tlsCfg,
	}
}

// Do  sent http request to url and save response
func (h *httpRequest) Do(ctx context.Context) error {

	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)

	scheme := "http"
	if h.tlsCfg != nil {
		scheme = "https"
	}

	// new http client
	httpClient := utils.NewHTTPClient(h.timeout, h.tlsCfg)

	if h.header != nil {
		for k, v := range h.header {
			httpClient.SetRequestHeader(k, v)
		}
	}

	url := fmt.Sprintf("%s://%s", scheme, h.url)
	fFile := filepath.Join(h.fileDir, h.fileName)
	if err := utils.CreateDir(h.fileDir); err != nil {
		return err
	}

	err := httpClient.Download(ctx, url, fFile)
	if err != nil {
		logger.Warnf("fail querying %s: %s, continue", url, err)
		return err
	}

	return nil
}
