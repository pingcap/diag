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

package scraper

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/pingcap/diag/collector/log/parser"
)

const (
	seekLimit = 1024 * 1024 * 1024 // 1MB
)

// LogScraper scraps log files of components
type LogScraper struct {
	Paths []string  // paths of log files
	Start time.Time // start time
	End   time.Time // end time
}

// NewLogScraper creates a new LogScraper
func NewLogScraper(l []string) *LogScraper {
	s := &LogScraper{
		Paths: make([]string, 0),
	}
	s.Paths = append(s.Paths, l...)
	return s
}

// Scrap implements the Scraper interface
func (s *LogScraper) Scrap(result *Sample) error {
	if result.Log == nil {
		result.Log = make(FileStat)
	}
	fileList := make([]string, 0)

	// extend all file paths
	for _, fp := range s.Paths {
		if fm, err := filepath.Glob(fp); err == nil {
			fileList = append(fileList, fm...)
		} else {
			fmt.Fprintf(os.Stderr, "error scrapping %s: %s\n", fp, err)
			continue
		}
	}

	// filter log files
	for _, fp := range fileList {
		if fi, err := os.Stat(fp); err == nil {
			if fi.IsDir() {
				continue
			}
			// check log content to filter by scrap time range
			in, err := logInRange(fp, fi, s.Start, s.End)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error parsing %s: %s\n", fi.Name(), err)
				continue
			}
			if !in {
				continue
			}

			result.Log[fp] = fi.Size()
		} else {
			fmt.Fprintf(os.Stderr, "error checking %s: %s\n", fi.Name(), err)
		}
	}

	return nil
}

func logInRange(fname string, fi fs.FileInfo, start, end time.Time) (bool, error) {
	f, err := os.Open(fname)
	if err != nil {
		return false, err
	}
	defer f.Close()

	bufr := bufio.NewReader(f)

	// read the first line of log file
	head, _, err := bufr.ReadLine()
	if err != nil {
		return false, err
	}

	ht := parseLine(head)
	if ht != nil && ht.After(end) {
		return false, nil
	}

	if fi.ModTime().Before(start) {
		return false, nil
	}

	return true, nil
}

func parseLine(line []byte) *time.Time {
	for _, p := range parser.List() {
		if t, _ := p.ParseHead(line); t != nil {
			return t
		}
	}
	return nil
}
