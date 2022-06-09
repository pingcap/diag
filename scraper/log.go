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
	"strings"
	"syscall"
	"time"

	"github.com/pingcap/diag/collector/log/parser"
)

const (
	seekLimit = 1024 * 1024 * 1024 // 1MB
)

// LogScraper scraps log files of components
type LogScraper struct {
	Paths []string        // paths of log files
	Types map[string]bool // log type
	Start time.Time       // start time
	End   time.Time       // end time
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

			logtype, in, err := getLogStatus(fp, fi, s.Start, s.End)
			if s.Types[logtype] && in {
				result.Log[fp] = fi.Size()
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "error checking %s: %s\n", fi.Name(), err)
			}

		} else {
			fmt.Fprintf(os.Stderr, "error checking %s: %s\n", fi.Name(), err)
		}
	}

	return nil
}

func getLogStatus(fpath string, fi fs.FileInfo, start, end time.Time) (logtype string, inrange bool, err error) {
	// collect stderr log despite time range
	if strings.Contains(filepath.Base(fpath), "stderr") {
		return "std", true, nil
	}

	f, err := os.Open(fpath)
	if err != nil {
		return "", false, err
	}
	defer f.Close()

	bufr := bufio.NewReader(f)
	// read the first line of log file
	head, _, err := bufr.ReadLine()
	if err == nil {
		ht := parseLine(head, parser.ListStd())
		if ht != nil {
			if ht.After(end) || fi.ModTime().Before(start) {
				return "std", false, nil
			}
			return "std", true, nil
		}
		p := &parser.SlowQueryParser{}
		ht, _ = p.ParseHead(head)
		if ht != nil {
			if ht.After(end) || fi.ModTime().Before(start) {
				return "slow", false, nil
			}
			return "slow", true, nil
		}
	}

	// use create time as head time for unknown file
	cTime := fi.Sys().(*syscall.Stat_t).Ctim
	ht := time.Unix(int64(cTime.Sec), int64(cTime.Nsec))
	if ht.After(end) || fi.ModTime().Before(start) {
		return "unknown", false, nil
	}
	return "unknown", true, nil
}

func parseLine(line []byte, parsers []parser.Parser) *time.Time {
	for _, p := range parsers {
		if t, _ := p.ParseHead(line); t != nil {
			return t
		}
	}
	return nil
}
