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

import "github.com/pingcap/tidb-foresight/scraper"

// Scrap run scrapers as Option configured
func Scrap(opt *scraper.Option) (*scraper.Sample, error) {
	scrapers := make([]scraper.Scrapper, 0)

	if len(opt.ConfigPaths) > 0 {
		scrapers = append(scrapers, scraper.NewConfigFileScraper(opt.ConfigPaths))
	}

	result := &scraper.Sample{}
	for _, s := range scrapers {
		if err := s.Scrap(result); err != nil {
			return result, err
		}
	}

	return result, nil
}
