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

package proto

import (
	"encoding/json"
	"os"
	"testing"
)

func TestJsonDecodeTikvConfigData(t *testing.T) {
	cfgData, err := os.ReadFile("../testdata/tikv.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg := NewTikvConfigData()
	if err := json.Unmarshal(cfgData, cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.LogLevel) == 0 {
		t.Fatal("wrong json decode result")
	}
	if cfg.Gc.EnableCompactionFilter == false {
		t.Fatal("wrong json decode result")
	}
}
