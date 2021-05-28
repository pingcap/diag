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
	"os"
	"path/filepath"
)

// ConfigFileScraper scraps configuration files of components
type ConfigFileScraper []string

// NewConfigFileScraper creates a new ConfigFileScraper
func NewConfigFileScraper(l []string) *ConfigFileScraper {
	c := make(ConfigFileScraper, 0)
	for _, f := range l {
		c = append(c, f)
	}
	return &c
}

// Scrap implements the Scraper interface
func (s *ConfigFileScraper) Scrap(result *Sample) error {
	if result.Config == nil {
		result.Config = make(FileStat)
	}
	fileList := make([]string, 0)

	// extend all file paths
	for _, fp := range *s {
		if fm, err := filepath.Glob(fp); err == nil {
			fileList = append(fileList, fm...)
		} else {
			return err
		}
	}

	// get file stats
	for _, fp := range fileList {
		if fi, err := os.Stat(fp); err == nil {
			if fi.IsDir() {
				continue
			}
			result.Config[fp] = fi.Size()
		} else {
			return err
		}
	}

	return nil
}
