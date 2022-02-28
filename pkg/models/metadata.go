// Copyright 2022 PingCAP, Inc.
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

package models

import (
	"github.com/pingcap/tiup/pkg/set"
)

// Some predefined key names for extended metadata
const (
	AttrKeyConfigFileList = "config_files"
	AttrKeyLogFileList    = "log_files"
	AttrKeyMetricsDir     = "metrics_dir"
)

// AttributeMap are extended key-values related to an object
type AttributeMap map[string]interface{}

// AddConfigFile inserts path of a config file to the list
func (attr AttributeMap) AddConfigFile(p string) {
	var tmp []string
	if m, ok := attr[AttrKeyConfigFileList]; ok {
		tmp = m.([]string)
	} else {
		tmp = make([]string, 0)
	}
	s := set.NewStringSet(tmp...)
	s.Insert(p)
	attr[AttrKeyConfigFileList] = s.Slice()
}

// AddLogFile inserts path of a log file to the list
func (attr AttributeMap) AddLogFile(p string) {
	var tmp []string
	if m, ok := attr[AttrKeyLogFileList]; ok {
		tmp = m.([]string)
	} else {
		tmp = make([]string, 0)
	}
	s := set.NewStringSet(tmp...)
	s.Insert(p)
	attr[AttrKeyLogFileList] = s.Slice()
}

// SetMetricsDir sets the path of metrics files
func (attr AttributeMap) SetMetricsDir(p string) {
	attr[AttrKeyMetricsDir] = p
}
