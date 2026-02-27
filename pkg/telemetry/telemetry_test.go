// Copyright 2020 PingCAP, Inc.
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

package telemetry

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	json "github.com/json-iterator/go"
	"github.com/pingcap/tiup/pkg/utils"
	"github.com/stretchr/testify/require"
)

func TestReport(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dst, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}

		msg := new(Report)
		err = json.Unmarshal(dst, msg)
		if err != nil {
			w.WriteHeader(400)
			return
		}

		if msg.UUID == "" {
			w.WriteHeader(400)
			return
		}
	}))

	defer ts.Close()

	tele := NewTelemetry()
	tele.cli = &utils.HTTPClient{}
	tele.cli.WithClient(ts.Client())
	tele.url = ts.URL

	msg := new(Report)

	err := tele.Report(context.TODO(), msg)
	require.Error(t, err)

	msg.UUID = "dfdfdf"
	err = tele.Report(context.TODO(), msg)
	require.NoError(t, err)
}
