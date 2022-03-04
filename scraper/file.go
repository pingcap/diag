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
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileScraper scraps normal files of components
type FileScraper struct {
	Paths []string  // paths of log files
	Start time.Time // start time
	End   time.Time // end time
}

// NewFlieScraper creates a new FileScraper
func NewFileScraper(l []string) *FileScraper {
	s := &FileScraper{
		Paths: make([]string, 0),
	}
	s.Paths = append(s.Paths, l...)
	return s
}

// Scrap implements the Scraper interface
func (s *FileScraper) Scrap(result *Sample) error {
	if result.File == nil {
		result.File = make(FileStat)
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
			if fi.ModTime().After(s.End) || fi.ModTime().Before(s.Start) {
				continue
			}

			result.File[fp] = fi.Size()
		} else {
			fmt.Fprintf(os.Stderr, "error checking %s: %s\n", fi.Name(), err)
		}
	}

	return nil
}
