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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pingcap/diag/scraper"
	"github.com/stretchr/testify/require"
)

const (
	scrapBegin = "2024-01-01T10:15:00+08:00"
	scrapEnd   = "2024-01-01T11:00:00+08:00"
)

const scrapTextLog = `[2024/01/01 10:00:00.000 +08:00][5][INFO] before
[2024/01/01 10:30:00.000 +08:00][5][INFO] inside
[2024/01/01 12:00:00.000 +08:00][5][INFO] after
`

const scrapJSONLog = `{"time":"2024/01/01 10:00:00.000 +08:00","level":"INFO","caller":"db/x.cc:1","message":"before"}
{"time":"2024/01/01 10:30:00.000 +08:00","level":"INFO","caller":"db/x.cc:1","message":"inside"}
{"time":"2024/01/01 12:00:00.000 +08:00","level":"INFO","caller":"db/x.cc:1","message":"after"}
`

func writeLog(t *testing.T, dir, name, content string) string {
	t.Helper()
	fp := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(fp, []byte(content), 0o644))
	return fp
}

func onlyKey(t *testing.T, m scraper.FileStat) string {
	t.Helper()
	require.Len(t, m, 1)
	for k := range m {
		return k
	}
	return ""
}

// TestScrapForwardsTrimOptions covers the layer the scraper package can not:
// Scrap is what turns the parsed flags into a LogScraper, so dropping Trim or
// TrimDir there would leave every scraper level test green while diag silently
// collects the files in full again.
func TestScrapForwardsTrimOptions(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		expected string
	}{
		{"text", scrapTextLog, "[2024/01/01 10:30:00.000 +08:00][5][INFO] inside\n"},
		{"json", scrapJSONLog, `{"time":"2024/01/01 10:30:00.000 +08:00","level":"INFO","caller":"db/x.cc:1","message":"inside"}` + "\n"},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			assert := require.New(t)
			dir, out := t.TempDir(), t.TempDir()
			src := writeLog(t, dir, "rocksdb.info", cs.content)

			sample, err := Scrap(&scraper.Option{
				LogPaths: []string{filepath.Join(dir, "*")},
				LogTypes: map[string]bool{scraper.LogTypeRocksDB: true},
				Start:    scrapBegin,
				End:      scrapEnd,
				Trim:     true,
				TrimDir:  out,
			})
			assert.NoError(err)

			target := onlyKey(t, sample.Log)
			assert.NotEqual(src, target, "the trimmed copy has to be reported, not the original file")
			assert.True(strings.HasPrefix(target, out), target)
			assert.Equal(scraper.LogTypeRocksDB, sample.LogTypes[target])

			content, err := os.ReadFile(target)
			assert.NoError(err)
			assert.Equal(cs.expected, string(content))
			assert.Equal(int64(len(cs.expected)), sample.Log[target],
				"the reported size is the size that gets downloaded")
		})
	}
}

// TestScrapWithoutTrimReportsTheOriginal is the other side of the forwarding:
// without --trim the original file must be reported.
func TestScrapWithoutTrimReportsTheOriginal(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()
	src := writeLog(t, dir, "rocksdb.info", scrapTextLog)

	sample, err := Scrap(&scraper.Option{
		LogPaths: []string{filepath.Join(dir, "*")},
		LogTypes: map[string]bool{scraper.LogTypeRocksDB: true},
		Start:    scrapBegin,
		End:      scrapEnd,
	})
	assert.NoError(err)

	target := onlyKey(t, sample.Log)
	assert.Equal(src, target)
	assert.Equal(int64(len(scrapTextLog)), sample.Log[target])
}
