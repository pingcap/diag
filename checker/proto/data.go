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
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/lensesio/tableprinter"
	"github.com/pingcap/diag/collector"
	"github.com/pingcap/tiup/pkg/logger/log"
)

const (
	PdComponentName                   = "PdConfig"
	TidbComponentName                 = "TidbConfig"
	TikvComponentName                 = "TikvConfig"
	TiflashComponentName              = "TiflashConfig"
	PerformanceDashboardComponentName = "performance.dashboard"
	ConfigType                        = "Config"
	PerformanceType                   = "performance"
)

type SourceDataV2 struct {
	ClusterInfo   *collector.ClusterJSON
	TidbVersion   string
	NodesData     map[string][]Config // {"component": {config, config, config, nil}}
	DashboardData *DashboardData
}

func (sd *SourceDataV2) AppendConfig(cfg Config, component string) {
	if n, ok := sd.NodesData[component]; ok {
		n = append(n, cfg)
		sd.NodesData[component] = n
	} else {
		sd.NodesData[component] = []Config{cfg}
	}
}

// todo@toto add check nil and how to format nil result
type Data interface {
	ActingName() string
}

type OutputMetaData struct {
	ClusterID   string
	ClusterName string
	TidbVersion string
	ActionID    string
	SampleData  Sample
}

type RuleResult struct {
	RuleName     string
	RuleID       int64
	Variation    string
	AlertingRule string
	InfoList     PrintTemplate // todo init
	Suggestion   string
}

type HandleData struct {
	UqiTag string
	Data   []Data
}

type PrintTemplate interface {
	// TemplateName() string
	CollectResult(*HandleData, interface{}) error
	Print(io.Writer)
}

type ConfPrintTemplate struct {
	Rule     *Rule
	InfoList []*ConfInfo
}
type ConfInfo struct {
	UniTag      string
	Val         string
	CheckResult string
}

func NewConfPrintTemplate(rule *Rule) *ConfPrintTemplate {
	return &ConfPrintTemplate{
		Rule: rule,
	}
}

func (c *ConfPrintTemplate) CollectResult(hd *HandleData, retValue interface{}) error {
	// rule valpath
	// find data val, host ip component warnlevel ......
	// fmt print and into instance checkinfo
	checkPass, _ := retValue.(bool)
	checkResult := ""
	if checkPass {
		checkResult = "OK"
	} else {
		checkResult = c.Rule.WarnLevel
	}
	valstr := c.GetValStr(hd)
	confInfo := &ConfInfo{
		UniTag:      hd.UqiTag,
		Val:         valstr,
		CheckResult: checkResult,
	}
	c.InfoList = append(c.InfoList, confInfo)
	return nil
}

func (c *ConfPrintTemplate) GetValStr(hd *HandleData) string {
	componentVal := c.SplitComponentAndPath(c.Rule.Variation)
	valmap := []string{}
	for _, data := range hd.Data {
		conf, ok := data.(Config)
		if !ok {
			log.Errorf("can't convert into config type", data.ActingName())
			continue
		}
		valpaths := componentVal[conf.GetComponent()]
		for _, valpath := range valpaths {
			if len(valpath) != 0 {
				rv := conf.GetValueByTagPath(valpath)
				valmap = append(valmap, fmt.Sprintf("%s.%s:%v", conf.GetComponent(), valpath, rv))
			}
		}
	}
	return strings.Join(valmap, ",")
}

func (c *ConfPrintTemplate) SplitComponentAndPath(varaition string) map[string][]string {
	valpaths := strings.Split(c.Rule.Variation, ",")
	componentVal := make(map[string][]string)
	for _, valpath := range valpaths {
		nn := strings.Split(valpath, ".")
		if len(nn) < 2 {
			continue
		}
		valsplit := nn[1:]
		val := strings.Join(valsplit, ".")
		if vals, ok := componentVal[nn[0]]; ok {
			vals = append(vals, val)
			componentVal[nn[0]] = vals
		} else {
			componentVal[nn[0]] = []string{val}
		}
	}
	return componentVal
}

func (c *ConfPrintTemplate) Print(out io.Writer) {
	printer := tableprinter.New(out)
	for _, rr := range c.InfoList {
		row, nums := tableprinter.StructParser.ParseRow(reflect.ValueOf(rr))
		printer.RenderRow(row, nums)
	}
}

type SqlPerformancePrintTemplate struct {
	Rule     *Rule
	InfoList *SqlPerformanceInfo
}

type SqlPerformanceInfo struct {
	NumDigest string
	Info      string
}

func NewSqlPerformancePrintTemplate(rule *Rule) *SqlPerformancePrintTemplate {
	return &SqlPerformancePrintTemplate{
		InfoList: &SqlPerformanceInfo{
			Info: "Please check the collect csv file for specific information",
		},
	}
}

func (c *SqlPerformancePrintTemplate) CollectResult(hd *HandleData, retValue interface{}) error {
	data, ok := hd.Data[0].(*DashboardData)
	if !ok {
		log.Errorf("convert into dashboarddata failed, ", data.ActingName())
	}
	switch c.Rule.Name {
	case "poor_execution_plan":
		checkResult, ok := retValue.(int)
		if !ok {
			return fmt.Errorf("retValue to int failed, %v", retValue)
		}
		c.InfoList.NumDigest = fmt.Sprintf("%d Digest trigger cordon", checkResult)
	case "old_version_count":
		checkResult, ok := retValue.(bool)
		if !ok {
			return fmt.Errorf("retValue to bool failed, %v", retValue)
		}
		if !checkResult {
			c.InfoList.NumDigest = fmt.Sprintf("%d Digest trigger cordon", 0)
		} else {
			c.InfoList.NumDigest = fmt.Sprintf("%d Digest trigger cordon", data.OldVersionProcesskey.Count)
		}
	case "scan_key_skip":
		c.InfoList.NumDigest = fmt.Sprintf("%d Digest trigger cordon", data.TombStoneStatistics.Count)
	}
	return nil
}

func (c *SqlPerformancePrintTemplate) Print(out io.Writer) {
	printer := tableprinter.New(out)
	row, nums := tableprinter.StructParser.ParseRow(reflect.ValueOf(c.InfoList))
	printer.RenderRow(row, nums)
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
	ActingName() string
	// GetValueByTagPath is used in gengine
	GetValueByTagPath(tagPath string) reflect.Value
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
	CheckType    string `yaml:"check_type" toml:"check_type"`
	ExpectRes    string `yaml:"expect_res" toml:"expect_res"`
	WarnLevel    string `yaml:"warn_level" toml:"warn_level"`
	Variation    string `yaml:"variation" toml:"variation"` // e.g. tidb.file.max_days,
	AlertingRule string `yaml:"alerting_rule" toml:"alerting_rule"`
	Suggestion   string `yaml:"suggestion" toml:"suggestion"`
}

type RuleSet map[string]*Rule // e.g {"Config": {"TidbConfigData": {&Rule{}, &Rule{}}}, "Dashboard": {}}

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
