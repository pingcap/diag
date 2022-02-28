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
	"reflect"
	"sort"
	"testing"
)

func TestAttributesMap(t *testing.T) {
	attr1 := AttributeMap{}
	attr1["foo"] = "bar"
	attr1.AddConfigFile("/path/to/1.conf")
	attr1.AddConfigFile("rel/path/to/2.conf")
	attr1.AddLogFile("1.log")
	attr1.AddLogFile("path/to/2.log")
	attr1.SetMetricsDir("some/metrics")
	sort.Strings(attr1[AttrKeyConfigFileList].([]string))
	sort.Strings(attr1[AttrKeyLogFileList].([]string))

	attr2 := AttributeMap{}
	attr2["foo"] = "bar"
	attr2[AttrKeyConfigFileList] = []string{
		"/path/to/1.conf",
		"rel/path/to/2.conf",
	}
	attr2[AttrKeyLogFileList] = []string{
		"1.log",
		"path/to/2.log",
	}
	sort.Strings(attr2[AttrKeyConfigFileList].([]string))
	sort.Strings(attr2[AttrKeyLogFileList].([]string))
	attr2[AttrKeyMetricsDir] = "some/metrics"

	if !reflect.DeepEqual(attr1, attr2) {
		t.Errorf("attributes mismatch:\n  added by func: %s\n  added by hand: %s\n", attr1, attr2)
	}
}
