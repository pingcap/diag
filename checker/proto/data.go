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
	"reflect"

	"github.com/Masterminds/semver"
	"github.com/pingcap/diag/collector"
)

const (
	PdComponentName                   = "PdConfig"
	TidbComponentName                 = "TidbConfig"
	TikvComponentName                 = "TikvConfig"
	TiflashComponentName              = "TiflashConfig"
	PerformanceDashboardComponentName = "performance.dashboard"
)

type SourceDataV2 struct {
	ClusterInfo   *collector.ClusterJSON
	TidbVersion   string
	NodesData     map[string][]Config // {"component": {config, config, config, nil}}
	DashboardData DashboardData
}

func (sd *SourceDataV2) AppendConfig(cfg Config, component string) {
	if n, ok := sd.NodesData[component]; ok {
		n = append(n, cfg)
		sd.NodesData[component] = n
	} else {
		sd.NodesData[component] = []Config{cfg}
	}
}

type OutputData struct {
	ClusterID   string
	ClusterName string
	TidbVersion string
	ActionID    string
	SampleData  Sample
	NodesData   []RuleResult // rule name
}

type RuleResult struct {
	RuleName      string
	RuleID        int64
	Variation     string
	AlertingRule  string
	DeployResults []DeployResult // todo init
	Suggestion    string
}

type DeployResult struct {
	ID    string `header:"node"`  // component_ip:port
	Value string `header:"value"` // name:val,name:val
	Res   string `header:"res"`   // warning ok info nodata
}

type Sample struct {
	SampleID      string
	SampleContent []string // e.g. {"Pd", "TiDB"....}
}

type NodeData struct {
	ID         string
	Timestamp  string
	Configs    []Config
	DeviceData DeviceData
}

type Config interface {
	GetComponent() string
	GetPort() int
	GetHost() string
	CheckNil() bool
	// GetValueByTagPath is used in gengine
	GetValueByTagPath(tagPath string) reflect.Value
}

type DashboardData struct {
	ExecutionPlanInfoList     map[string][2]ExecutionPlanInfo // e.g {"xsdfasdf22sdf": {{min_execution_info}, {max_execution_info}}}
	OldVersionProcesskeyCount struct {
		GcLifeTime int // s
		Count      int
	}
	TombStoneStatistics struct {
		Count int
	}
}

type ExecutionPlanInfo struct {
	PlanDigest     string
	MaxLastTime    int64
	AvgProcessTime int64 // ms
}

type DeviceData struct{}

// ruletag: checkType, datatype, component
type Rule struct {
	// version
	ID           int64  `yaml:"id" toml:"id"`
	Name         string `yaml:"name" toml:"name"`
	Description  string `yaml:"description" toml:"description"`
	ExecuteRule  string `yaml:"execute_rule" toml:"execute_rule"`
	NameStruct   string `yaml:"name_struct" toml:"name_struct"` // datatype.component
	ExpectRes    string `yaml:"expect_res" toml:"expect_res"`
	WarnLevel    string `yaml:"warn_level" toml:"warn_level"`
	Variation    string `yaml:"variation" toml:"variation"` // e.g. tidb.file.max_days,
	AlertingRule string `yaml:"alerting_rule" toml:"alerting_rule"`
	Suggestion   string `yaml:"suggestion" toml:"suggestion"`
}

type RuleSet map[string][]*Rule // e.g {"Config": {"TidbConfigData": {&Rule{}, &Rule{}}}, "Dashboard": {}}

type VersionRange string

func (vr VersionRange) Contain(target string) (bool, error) {
	if len(vr) == 0 {
		return true, nil
	}
	verCheck, err := semver.NewConstraint(string(vr))
	if err != nil {
		return false, err
	}
	ver, err := semver.NewVersion(target)
	if err != nil {
		return false, err
	}
	return verCheck.Check(ver), nil
}
