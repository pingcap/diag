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

// Option is configurations of how scrapper works
type Option struct {
	LogPaths    []string // paths of log files
	ConfigPaths []string // paths of config files
	Start       string   // start time
	End         string   // end time
}

// FileStat is the size information of a file to scrap
// map: filename (full path) -> file size (bytes)
type FileStat map[string]int64

// Sample is the result of scrapping
type Sample struct {
	Log    FileStat `json:"log_files,omitempty"`
	Config FileStat `json:"config_files,omitempty"`
}

// Scrapper is used to scrap a kind of files
type Scrapper interface {
	Scrap(*Sample) error
}
