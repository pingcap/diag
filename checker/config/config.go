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

package config

import (
	_ "embed"
	"github.com/BurntSushi/toml"
	"github.com/pingcap/diag/checker/proto"
)

// TODO load all config data at once may not the best solution.
//
//go:embed rule_beta.toml
var betaRuleStr string

type RuleItem struct {
	proto.Rule `yaml:",inline"`
	Version    proto.VersionRange `yaml:"version" toml:"version"`
}

type RuleSpec struct {
	Rule []RuleItem `yaml:"rule" toml:"rule"`
}

func (rs *RuleSpec) FilterOnVersion(ver string) (proto.RuleSet, error) {
	rSet := proto.RuleSet{}
	for idx := range rs.Rule {
		ok, err := rs.Rule[idx].Version.Contain(ver)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		// set match rule to rSet
		rulename := rs.Rule[idx].Rule.Name
		rSet[rulename] = &rs.Rule[idx].Rule
	}
	return rSet, nil
}

type FilterFunc func(item RuleItem) (bool, error)

func (rs *RuleSpec) FilterOn(filter FilterFunc) (proto.RuleSet, error) {
	rSet := proto.RuleSet{}
	for idx := range rs.Rule {
		ok, err := filter(rs.Rule[idx])
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		// set match rule to rSet
		rulename := rs.Rule[idx].Rule.Name
		rSet[rulename] = &rs.Rule[idx].Rule
	}
	return rSet, nil
}

func LoadBetaRuleSpec() (*RuleSpec, error) {
	ruleSpec := &RuleSpec{Rule: []RuleItem{}}
	if _, err := toml.Decode(betaRuleStr, ruleSpec); err != nil {
		return nil, err
	}
	return ruleSpec, nil
}
