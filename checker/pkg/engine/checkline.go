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
	"go.uber.org/zap"

	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/log"
)

type CheckLine interface {
	Check(data proto.SourceDataV2, rule proto.RuleSet) error
}

const resultlen int = 10

type RuleCheck struct {
	Result chan map[string]proto.RuleResult
	//close       chan struct{}
}

func (c *RuleCheck) Init() {
	c.Result = make(chan map[string]proto.RuleResult, resultlen)
}

func (c *RuleCheck) GetResultChan() <-chan map[string]proto.RuleResult {
	return c.Result
}

func (c *RuleCheck) Check(data proto.SourceDataV2, rule proto.RuleSet) error {
	defer close(c.Result)
	for component, configs := range data.NodesData {
		configRule, ok := rule[component]
		if !ok {
			log.Error("no such component rule, ", zap.String("component", component))
			return errors.New("no rule found")
		}
		ruleSpread := make(map[string]proto.RuleResult) // todo check pointer
		for _, config := range configs {
			checkunit := &Rule{}
			err := checkunit.Mapping(config, configRule, proto.DeviceData{}, ruleSpread) // todo delete device
			if err != nil {
				log.Error("mapping failed", zap.Error(err))
				return err
			}
		}
		c.Result <- ruleSpread
	}
	return nil
}
