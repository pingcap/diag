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
	"testing"

	"github.com/pingcap/diag/scraper"
	"github.com/stretchr/testify/require"
)

// resetCLI puts the flag state of the shared root command back to its initial
// value. cobra keeps the values it parsed between Execute calls, so a second
// run of this package - `go test -count=2`, or a shuffled order - would
// otherwise inherit --trim-dir from the test that ran before it and the
// validation under test would not trigger. The struct is reset in place because
// the flags hold pointers into it.
func resetCLI() {
	*opt = scraper.Option{
		LogPaths:    []string{},
		ConfigPaths: []string{},
		FilePaths:   []string{},
		LogTypes:    map[string]bool{},
	}
}

// TestCLIRejectsTrimWithoutTrimDir runs the cobra command instead of calling
// the scraper directly: this validation lives in the command, so only executing
// it proves the check is still wired. Without it the scraper would receive a
// bare --trim and the collection would abort.
func TestCLIRejectsTrimWithoutTrimDir(t *testing.T) {
	assert := require.New(t)
	resetCLI()
	dir := t.TempDir()
	writeLog(t, dir, "rocksdb.info", scrapTextLog)

	rootCmd.SetArgs([]string{
		"--log", filepath.Join(dir, "*"),
		"-f", scrapBegin,
		"-t", scrapEnd,
		"--logtype", "rocksdb",
		"--trim",
	})

	err := rootCmd.Execute()
	assert.Error(err)
	assert.Contains(err.Error(), "--trim-dir is required")
}

// TestCLIAcceptsTrimWithTrimDir is the positive case: the flags together must
// get through validation.
func TestCLIAcceptsTrimWithTrimDir(t *testing.T) {
	assert := require.New(t)
	resetCLI()
	dir, out := t.TempDir(), t.TempDir()
	writeLog(t, dir, "rocksdb.info", scrapTextLog)

	rootCmd.SetArgs([]string{
		"--log", filepath.Join(dir, "*"),
		"-f", scrapBegin,
		"-t", scrapEnd,
		"--logtype", "rocksdb",
		"--trim",
		"--trim-dir", out,
	})

	assert.NoError(rootCmd.Execute())

	entries, err := os.ReadDir(out)
	assert.NoError(err)
	assert.NotEmpty(entries, "the trimmed copy has to be written")
}
