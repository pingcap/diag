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

package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bilibili/gengine/builder"
	"github.com/bilibili/gengine/context"
	"github.com/bilibili/gengine/engine"
	"github.com/pingcap/diag/checker/pkg/utils"
	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/log"
	"go.uber.org/zap"
)

type Rule struct {
	Component string
	Node      string
	Port      int
}

func (r *Rule) Mapping(sd proto.Config, rule []*proto.Rule, deviceData proto.DeviceData, rr map[string]proto.RuleResult) error {
	dataContext := context.NewDataContext()
	if sd.CheckNil() {
		r.FormatNoDataResult(rule, sd, rr)
	}
	dataContext.Add("config", sd)
	dataContext.Add("DeviceData", deviceData)
	dataContext.Add("ToInt", utils.ValueToInt)
	dataContext.Add("ToString", utils.ValueToString)
	dataContext.Add("ToBool", utils.ValueToBool)
	dataContext.Add("ToFloat", utils.ValueToFloat)
	dataContext.Add("FlatMap", utils.FlatMap)
	dataContext.Add("ElemInRange", utils.ElemInRange)
	ruleBuilder := builder.NewRuleBuilder(dataContext)
	rulestr := r.FormatString(rule)
	if err := ruleBuilder.BuildRuleFromString(rulestr); err != nil {
		log.Error("build rule %+v err:%s ", zap.Any("rule", rule), zap.Error(err))
		return err
	}
	eng := engine.NewGengine()
	if err := eng.ExecuteConcurrent(ruleBuilder); err != nil {
		log.Error("execute rule error: %v", zap.Error(err))
		return err
	}
	result, _ := eng.GetRulesResultMap()
	err := r.FormatResultV2(result, rule, sd, rr)
	if err != nil {
		return err
	}
	return nil
}

func (r *Rule) FormatString(rules []*proto.Rule) string {
	rulestr := ""
	for _, rule := range rules {
		rulestr += rule.ExecuteRule + "\n"
	}
	return rulestr
}

func (r *Rule) FormatNoDataResult(rules []*proto.Rule, config proto.Config, rr map[string]proto.RuleResult) error {
	for _, rule := range rules {
		res := proto.DeployResult{
			ID:    fmt.Sprintf("%s_%s:%d", config.GetComponent(), config.GetHost(), config.GetPort()),
			Value: r.GetValue(rule.Variation, config),
			Res:   "NoData",
		}
		ds, ok := rr[rule.Name]
		if ok {
			ds.DeployResults = append(ds.DeployResults, res)
		} else {
			ds := proto.RuleResult{
				RuleName:     rule.Name,
				RuleID:       rule.ID,
				Variation:    rule.Variation,
				AlertingRule: rule.AlertingRule,
				Suggestion:   rule.Suggestion,
			}
			ds.DeployResults = []proto.DeployResult{res}
			rr[rule.Name] = ds
		}
	}
	return nil
}

func (r *Rule) FormatResultV2(rawresult map[string]interface{}, rules []*proto.Rule, config proto.Config, rr map[string]proto.RuleResult) error {
	for _, rule := range rules {
		val, ok := rawresult[rule.Name]
		if !ok {
			log.Error("no rule result mapping,", zap.String("name", rule.Name))
			return errors.New("no rule result mapping")
		}
		mappingRes, _ := val.(bool)
		var Pass string
		if mappingRes {
			Pass = "OK"
		} else {
			Pass = rule.WarnLevel
		}
		res := proto.DeployResult{
			ID:    fmt.Sprintf("%s_%s:%d", config.GetComponent(), config.GetHost(), config.GetPort()),
			Value: r.GetValue(rule.Variation, config),
			Res:   Pass,
		}
		ds, ok := rr[rule.Name]
		if ok {
			ds.DeployResults = append(ds.DeployResults, res)
		} else {
			ds := proto.RuleResult{
				RuleName:     rule.Name,
				RuleID:       rule.ID,
				Variation:    rule.Variation,
				AlertingRule: rule.AlertingRule,
				Suggestion:   rule.Suggestion,
			}
			ds.DeployResults = []proto.DeployResult{res}
			rr[rule.Name] = ds
		}
	}
	return nil
}

func (r *Rule) GetValue(valpath string, config proto.Config) string {
	valpaths := strings.Split(valpath, ",")
	valmap := []string{}
	for _, valpath := range valpaths {
		if len(valpath) != 0 {
			rv := config.GetValueByTagPath(valpath) // empty will coredump
			valmap = append(valmap, fmt.Sprintf("%s.%s:%v", config.GetComponent(), valpath, rv))
		}
	}
	return strings.Join(valmap, ",")
}
