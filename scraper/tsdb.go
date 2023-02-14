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
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/pingcap/diag/pkg/utils"
)

// TSDBScraper scraps log files of components
type TSDBScraper struct {
	Paths []string  // paths of log files
	Start time.Time // start time
	End   time.Time // end time
}

type prometheusMeta struct {
	MinTime int64 `json:"minTime"`
	MaxTime int64 `json:"maxTime"`
}

// Scrap implements the Scraper interface
func (s *TSDBScraper) Scrap(result *Sample) error {
	if result.TSDB == nil {
		result.TSDB = make(FileStat)
	}

	dirEntrys, err := os.ReadDir(s.Paths[0])
	if err != nil {
		panic(err)
	}

	// filter tsdb block dirs
	for _, d := range dirEntrys {
		if !d.IsDir() {
			continue
		}
		dirPath := filepath.Join(s.Paths[0], d.Name())

		if d.Name() != "chunks_head" {
			data, err := os.ReadFile(filepath.Join(dirPath, "meta.json"))
			if err != nil {
				continue
			}

			var meta prometheusMeta
			err = json.Unmarshal(data, &meta)
			if err != nil {
				continue
			}

			// tsdb block is not in time range
			if meta.MaxTime < s.Start.UnixMilli() || meta.MinTime > s.End.UnixMilli() {
				continue
			}
		}

		now := time.Now()
		// ignore chunks_head
		if s.End.Before(now.Add(-3 * time.Hour)) {
			continue
		}

		size, err := utils.DirSize(dirPath)
		if err != nil {
			size = 0
		}
		result.TSDB[dirPath] = size

	}

	return nil
}
