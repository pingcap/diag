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
	"fmt"
	"os"

	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/diag/scraper"
)

// Scrap run scrapers as Option configured
func Scrap(opt *scraper.Option) (*scraper.Sample, error) {
	scrapers := make([]scraper.Scrapper, 0)

	if len(opt.ConfigPaths) > 0 {
		scrapers = append(scrapers, scraper.NewConfigFileScraper(opt.ConfigPaths))
	}
	if len(opt.LogPaths) > 0 {
		s := scraper.NewLogScraper(opt.LogPaths)
		var err error
		if s.Start, err = utils.ParseTime(opt.Start); err != nil {
			return nil, err
		}
		if s.End, err = utils.ParseTime(opt.End); err != nil {
			return nil, err
		}
		scrapers = append(scrapers, s)
	}

	if len(opt.FilePaths) > 0 {
		s := scraper.NewFileScraper(opt.FilePaths)
		var err error
		if s.Start, err = utils.ParseTime(opt.Start); err != nil {
			return nil, err
		}
		if s.End, err = utils.ParseTime(opt.End); err != nil {
			return nil, err
		}
		scrapers = append(scrapers, s)
	}

	result := &scraper.Sample{}
	for _, s := range scrapers {
		if err := s.Scrap(result); err != nil {
			fmt.Fprint(os.Stderr, err)
		}
	}

	return result, nil
}
