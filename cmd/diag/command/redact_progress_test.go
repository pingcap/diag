// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/pingcap/diag/pkg/redact"
	"github.com/stretchr/testify/require"
)

func TestRedactionProgressDetail(t *testing.T) {
	state := redact.Progress{Bytes: 50, TotalBytes: 100, Files: 1, TotalFiles: 2}
	require.Contains(t, redactionProgressDetail(state, 2*time.Second), "[=====     ]  50% | 1/2 files")
	require.NotContains(t, redactionProgressDetail(state, 2*time.Second), "MiB")
	state.Bytes = 100
	require.Contains(t, redactionProgressDetail(state, 0), "50%")
	state.Files = 2
	require.Contains(t, redactionProgressDetail(state, 0), "99%")
	state.Done = true
	require.Contains(t, redactionProgressDetail(state, 0), "[==========] 100%")
	require.Contains(t, redactionProgressDetail(redact.Progress{}, 0), "0% | 0/0 files")
	state = redact.Progress{Files: 1, TotalFiles: 2}
	require.Contains(t, redactionProgressDetail(state, 0), "50%")
}

func TestRedirectedRedactionProgress(t *testing.T) {
	var out bytes.Buffer
	p := &redactionProgress{writer: &out}
	p.update(redact.Progress{Phase: "Scanning IPs"})
	p.update(redact.Progress{Phase: "Scanning IPs", Bytes: 50, TotalBytes: 100})
	p.update(redact.Progress{Phase: "Scanning IPs", Bytes: 100, TotalBytes: 100, Done: true})
	p.close(nil)
	require.NotContains(t, out.String(), "\x1b")
	require.NotContains(t, out.String(), "\r")
	require.Equal(t, 2, bytes.Count(out.Bytes(), []byte("\n")))
	require.Contains(t, out.String(), "Scanning IPs done")
	out.Reset()
	p.update(redact.Progress{Phase: "Redacting"})
	p.close(errors.New("fixture failure"))
	require.Contains(t, out.String(), "IP redaction failed")
	require.NotContains(t, out.String(), "done")
}
