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
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/pingcap/diag/checker/pkg/utils"
	"os"
	"reflect"
	"testing"
)

func TestVersionRange_IsTarget(t *testing.T) {
	tt := []struct{
		Target string
		VRange VersionRange
		Expected bool
	}{
		{Target: "v5.0.1", VRange: VersionRange(">=v5.0.1,<v5.0.2"), Expected: true},
		{Target: "v5.0.1", VRange: VersionRange(">=v4.0.1,<v4.0.2"), Expected: false},
		{Target: "v5.0.1", VRange: VersionRange("<v4.0.2||>=v5.0"), Expected: true},
		{Target: "v5.0.1", VRange: VersionRange("v5.0.1"), Expected: true},
		{Target: "v5.0.1", VRange: VersionRange("v5.0.2"), Expected: false},
		{Target: "v5.0.1", VRange: VersionRange("v5.0.x"), Expected: true},
		{Target: "v5.0.1", VRange: VersionRange(">=v5.0.1"), Expected: true},
		{Target: "v5.0.1", VRange: VersionRange("<v5.1.1"), Expected: true},
		{Target: "v5.0.1", VRange: VersionRange("<v5.0.1"), Expected: false},
		{Target: "v5.0.1", VRange: VersionRange(">v5.1.1"), Expected: false},
	}
	for _, tc := range tt {
		t.Run("", func(t *testing.T) {
			if ok, err := tc.VRange.Contain(tc.Target); err != nil {
				t.Fatal(err)
			}else if ok != tc.Expected {
				t.Error("wrong result")
			}
		})
	}
}

func TestTomlDecodePdConfigData(t *testing.T) {
	cfgData := `
name = ""
lease = 0
[pd-server]
metric-storage = "http://127.0.0.1:9090"
[schedule]
max-merge-region-size = 1
enable-one-way-merge = true
leader-schedule-limit = 1
`
	cfg := NewPdConfigData()
	if _, err := toml.Decode(cfgData, cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Schedule.MaxMergeRegionSize == 0 {
		t.Fatal("wrong toml decode result")
	}
}

func TestJsonDecodePdConfigData(t *testing.T) {
	cfgData, err := os.ReadFile("../testdata/pd.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg := NewPdConfigData()
	if err := json.Unmarshal(cfgData, cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Schedule.MaxMergeRegionSize == 0 {
		t.Fatal("wrong json decode result")
	}
}

func TestJsonDecodeTidbConfigData(t *testing.T) {
	cfgData, err := os.ReadFile("../testdata/tidb.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg := NewTidbConfigData()
	if err := json.Unmarshal(cfgData, cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Log.File.MaxDays == 0 {
		t.Fatal("wrong json decode result")
	}
	if cfg.Log.File.MaxBackups == 0 {
		t.Fatal("wrong json decode result")
	}
}

func TestPdConfigData_GetValueByTagPath(t *testing.T) {
	cfg := NewPdConfigData()
	cfg.Log.File.MaxDays = 10
	cfg.Schedule.MaxSnapshotCount = 1
	cfg.Schedule.MaxPendingPeerCount = 1
	cfg.Schedule.StoreLimit = map[uint64]StoreLimitConfig{
		1:StoreLimitConfig{
			AddPeer:    10,
			RemovePeer: 11,
		},
	}
	cfg.Log.Level = "debug"
	tt := []struct{
		TagPath string
		Expect reflect.Value
	}{
		{
			TagPath: "log.file.max-days",
			Expect: reflect.ValueOf(10),
		},
		{
			TagPath: "schedule.max-snapshot-count",
			Expect: reflect.ValueOf(1),
		},
		{
			TagPath: "schedule.max-pending-peer-count",
			Expect: reflect.ValueOf(1),
		},
		{
			TagPath: "schedule.store-limit.1.add-peer",
			Expect: reflect.ValueOf(float64(10)),
		},
		{
			TagPath: "schedule.store-limit.1.remove-peer",
			Expect: reflect.ValueOf(float64(11)),
		},
		{
			TagPath: "log.level",
			Expect: reflect.ValueOf("debug"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.TagPath, func(t *testing.T) {
			result := cfg.GetValueByTagPath(tc.TagPath)
			if fmt.Sprint(result) != fmt.Sprint(tc.Expect){
				t.Errorf("wrong value for path: %+v", tc.TagPath)
			}
		})
	}
}

func TestTidbConfigData_GetValueByTagPath(t *testing.T) {
	cfg := NewTidbConfigData()
	cfg.Log.Level = "debug"
	cfg.TidbConfig.EnableStreaming = true
	cfg.Log.File.MaxDays = 10
	tt := []struct{
		TagPath string
		Expect reflect.Value
	}{
		{
			TagPath: "log.level",
			Expect: reflect.ValueOf("debug"),
		},
		{
			TagPath: "enable-streaming",
			Expect: reflect.ValueOf(true),
		},
		{
			TagPath: "log.file.max-days",
			Expect: reflect.ValueOf(10),
		},
	}
	for _, tc := range tt {
		t.Run(tc.TagPath, func(t *testing.T) {
			result := cfg.GetValueByTagPath(tc.TagPath)
			if fmt.Sprint(result) != fmt.Sprint(tc.Expect){
				t.Errorf("wrong value for path: %+v", tc.TagPath)
			}
		})
	}
}

func TestTidbConfigData_GetDeprecatedValueByTagPath(t *testing.T) {
	bs ,err := os.ReadFile("../testdata/tidb.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg := NewTidbConfigData()
	if err := json.Unmarshal(bs, cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.TidbConfig.EnableStreaming == false {
		t.Fatal("wrong")
	}
	result := cfg.GetValueByTagPath("enable-streaming")
	if fmt.Sprint(result) != fmt.Sprint(reflect.ValueOf(true)){
		t.Errorf("wrong")
	}
}

func TestTikvConfigData_GetValueByTagPath(t *testing.T) {
	cfg := NewTikvConfigData()
	cfg.LogLevel = "debug"
	cfg.Gc.EnableCompactionFilter = true
	tt := []struct{
		TagPath string
		Expect reflect.Value
	}{
		{
			TagPath: "log-level",
			Expect: reflect.ValueOf("debug"),
		},
		{
			TagPath: "gc.enable-compaction-filter",
			Expect: reflect.ValueOf(true),
		},
	}
	for _, tc := range tt {
		t.Run(tc.TagPath, func(t *testing.T) {
			result := cfg.GetValueByTagPath(tc.TagPath)
			if fmt.Sprint(result) != fmt.Sprint(tc.Expect){
				t.Errorf("wrong value for path: %+v", tc.TagPath)
			}
		})
	}
}

func TestPdConfigData_MaxFromMap(t *testing.T) {
	bs ,err := os.ReadFile("../testdata/pd.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg := NewPdConfigData()
	if err := json.Unmarshal(bs, cfg); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v\n",cfg.Schedule.StoreLimit)
	ok := utils.ElemInRange(utils.FlatMap(cfg.GetValueByTagPath("schedule.store-limit"),"add-peer"), 0, 100)
	if !ok {
		t.Errorf("wrong")
	}
	ok = utils.ElemInRange(utils.FlatMap(cfg.GetValueByTagPath("schedule.store-limit"),"remove-peer"), 0, 100)
	if !ok{
		t.Errorf("wrong")
	}
}