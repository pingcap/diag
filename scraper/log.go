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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/pingcap/tidb-foresight/log/parser"
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
			return err
		}
	}

	// filter log files
	for _, fp := range fileList {
		if fi, err := os.Stat(fp); err == nil {
			if fi.IsDir() {
				continue
			}
			// modify time is earlier than scrap start
			if fi.ModTime().Before(s.Start) {
				continue
			}
			// check log content to filter by scrap time range
			in, err := logInRange(fp, fi, s.Start, s.End)
			if err != nil {
				return err
			}
			if !in {
				continue
			}

			result.Log[fp] = fi.Size()
		} else {
			return err
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

	// return if the first line is later than scrap end
	ht := parseLine(head)
	if ht == nil {
		return false, fmt.Errorf("the first line is not a valid log line")
	}
	if ht.After(end) {
		return false, nil
	}

	// read the last line of log file
	tail := make([]byte, seekLimit)
	fsize := fi.Size()

	// Seek backwards for a line
	var cursor int64
	for cursor = 0; cursor > -seekLimit; cursor-- {
		// stop if reached the file begining
		if cursor == -fsize {
			break
		}

		// seek one byte backward
		f.Seek(cursor, io.SeekEnd)
		buf := make([]byte, 1)
		_, err := f.Read(buf)
		// ignore EOF, we are reading backwards
		if err != nil && !errors.Is(err, io.EOF) {
			return false, err
		}

		// stop if reached a line breaker
		// we set a 3 bytes threshold to avoid reading empty line at the end
		if cursor < -3 && (buf[0] == 10 || buf[0] == 13) {
			break
		}

		// populate the line from end to begin
		tail[seekLimit+cursor-1] = buf[0]
	}
	tail = tail[seekLimit+cursor:] // cut

	// return if the last line is before scrap start
	tt := parseLine(tail)
	if tt == nil {
		return false, fmt.Errorf("the last line is not a valid log line")
	}
	if tt.Before(start) {
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
