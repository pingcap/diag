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

package redact

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pingcap/diag/collector/log/parser"
	"github.com/stretchr/testify/require"
)

func TestQuestionMarksKeepDistinctPathsAndStructure(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	for _, dir := range []string{"10.1.1.1", "10.1.1.2", "monitor/metrics"} {
		require.NoError(t, os.MkdirAll(filepath.Join(input, dir), 0700))
	}
	write := func(name, body string) {
		require.NoError(t, os.WriteFile(filepath.Join(input, name), []byte(body), 0600))
	}
	write("cluster.json", `{"hosts":["10.1.1.1","10.1.1.2"],"log_files":["10.1.1.1/tidb.log","10.1.1.2/tidb.log"]}`)
	write("10.1.1.1/tidb.log", "peer=10.1.1.1:4000 remote=10.1.1.2 ipv6=[fd00::1]:4000 mapped=[::ffff:10.1.1.1] phone=13800000000\n")
	write("10.1.1.2/tidb.log", "peer=10.1.1.2:4000 sql='10.1.1.1'\n")
	write("10.1.1.1/tidb.toml", "host='10.1.1.1'\n")
	write("monitor/metrics/query.json", `{"instance":"10.1.1.1:4000","peer":"10.1.1.2:20160"}`)
	r, err := Discover(input)
	require.NoError(t, err)
	require.NoError(t, r.MaskLogIPv4())
	output, report, err := r.Copy(input, filepath.Join(root, "output"))
	require.NoError(t, err)
	require.Equal(t, 5, report.LogIPv4QuestionMarks)
	body, err := os.ReadFile(filepath.Join(output, "198.18.0.1/tidb.log"))
	require.NoError(t, err)
	require.Equal(t, "peer=?:4000 remote=? ipv6=[2001:db8::1]:4000 mapped=[?] phone=13800000000\n", string(body))
	body, err = os.ReadFile(filepath.Join(output, "198.18.0.2/tidb.log"))
	require.NoError(t, err)
	require.Equal(t, "peer=?:4000 sql='?'\n", string(body))
	body, err = os.ReadFile(filepath.Join(output, "cluster.json"))
	require.NoError(t, err)
	var metadata struct {
		Hosts []string `json:"hosts"`
		Logs  []string `json:"log_files"`
	}
	require.NoError(t, json.Unmarshal(body, &metadata))
	require.Equal(t, []string{"198.18.0.1", "198.18.0.2"}, metadata.Hosts)
	for _, name := range metadata.Logs {
		_, err := os.Stat(filepath.Join(output, name))
		require.NoError(t, err)
	}
	_, err = os.ReadFile(filepath.Join(output, "10.1.1.1/tidb.toml"))
	require.True(t, os.IsNotExist(err))
	body, err = os.ReadFile(filepath.Join(output, "198.18.0.1/tidb.toml"))
	require.NoError(t, err)
	require.Equal(t, "host='198.18.0.1'\n", string(body))
	body, err = os.ReadFile(filepath.Join(output, "monitor/metrics/query.json"))
	require.NoError(t, err)
	require.JSONEq(t, `{"instance":"198.18.0.1:4000","peer":"198.18.0.2:20160"}`, string(body))
	// Verification must reject even a known surrogate IPv4 in a log body.
	require.NoError(t, os.WriteFile(filepath.Join(output, "198.18.0.1/tidb.log"), []byte("peer=198.18.0.1"), 0600))
	require.ErrorContains(t, r.verify(output), "IPv4 literal remains")
}

func TestQuestionMarkCompressedAndArchivedLogs(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	require.NoError(t, os.MkdirAll(filepath.Join(input, "logs"), 0700))
	var compressed bytes.Buffer
	gw := gzip.NewWriter(&compressed)
	_, err := gw.Write([]byte("peer=10.1.1.1:4000\n"))
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	require.NoError(t, os.WriteFile(filepath.Join(input, "tidb.log.1.gz"), compressed.Bytes(), 0600))
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	entry, err := zw.Create("events.txt")
	require.NoError(t, err)
	_, err = entry.Write([]byte("peer=10.1.1.1:4000\n"))
	require.NoError(t, err)
	entry, err = zw.Create("cluster.json")
	require.NoError(t, err)
	_, err = entry.Write([]byte(`{"host":"10.1.1.1"}`))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, os.WriteFile(filepath.Join(input, "logs/bundle.zip"), archive.Bytes(), 0600))
	r, err := Discover(input)
	require.NoError(t, err)
	require.NoError(t, r.MaskLogIPv4())
	output, report, err := r.Copy(input, filepath.Join(root, "output"))
	require.NoError(t, err)
	require.Equal(t, 2, report.LogIPv4QuestionMarks)
	body, err := os.ReadFile(filepath.Join(output, "tidb.log.1.gz"))
	require.NoError(t, err)
	gr, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)
	defer gr.Close()
	body, err = io.ReadAll(gr)
	require.NoError(t, err)
	require.Equal(t, "peer=?:4000\n", string(body))
	body, err = os.ReadFile(filepath.Join(output, "logs/bundle.zip"))
	require.NoError(t, err)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	for _, file := range zr.File {
		reader, err := file.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		if file.Name == "events.txt" {
			require.Equal(t, "peer=?:4000\n", string(data))
		} else {
			require.JSONEq(t, `{"host":"198.18.0.1"}`, string(data))
		}
	}
}

func TestQuestionMarkLogClassificationAndSelection(t *testing.T) {
	for _, name := range []string{"tidb.log", "10.1.1.1/tidb_slow_query.log", "tidb.log.1", "tidb-audit.log.json", "logs/pod/output.txt", "logs/bundle.zip/events.txt", "db/LOG", "stderr.stderr", "cluster_audit/session.txt"} {
		require.True(t, isLogFile(name), name)
	}
	for _, name := range []string{"cluster.json", "logs/cluster.json", "logs/settings.toml", "meta.yaml", "monitor/metrics/tidb.log", "config/tidb.log", "db_vars/session.txt", "host/config.json"} {
		require.False(t, isLogFile(name), name)
	}
	r, err := New([]string{"10.1.1.1"})
	require.NoError(t, err)
	require.Error(t, r.MaskLogIPv4())
}

func TestQuestionMarksPreserveStandardAndSlowLogHeaders(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	require.NoError(t, os.Mkdir(input, 0700))
	standard := `[2026/10/21 10:00:00.000 +08:00] [INFO] [conn.go:1] ["peer"] [addr=10.1.1.1:4000]` + "\n"
	slow := "# Time: 2026-10-21T10:00:00+08:00\n# User@Host: root[root] @ 10.1.1.1 []\nSELECT '10.1.1.1';\n"
	require.NoError(t, os.WriteFile(filepath.Join(input, "tidb.log"), []byte(standard), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(input, "tidb_slow_query.log"), []byte(slow), 0600))
	r, err := Discover(input)
	require.NoError(t, err)
	require.NoError(t, r.MaskLogIPv4())
	output, _, err := r.Copy(input, filepath.Join(root, "output"))
	require.NoError(t, err)
	for _, tc := range []struct {
		name, original string
		parser         parser.Parser
	}{
		{"tidb.log", standard, &parser.UnifiedLogParser{}},
		{"tidb_slow_query.log", slow, &parser.SlowQueryParser{}},
	} {
		data, err := os.ReadFile(filepath.Join(output, tc.name))
		require.NoError(t, err)
		require.Contains(t, string(data), "?")
		beforeTime, beforeLevel := tc.parser.ParseHead(bytes.SplitN([]byte(tc.original), []byte("\n"), 2)[0])
		afterTime, afterLevel := tc.parser.ParseHead(bytes.SplitN(data, []byte("\n"), 2)[0])
		require.NotNil(t, beforeTime)
		require.Equal(t, beforeTime, afterTime)
		require.Equal(t, beforeLevel, afterLevel)
	}
}
