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

package engine_test

import (
	"github.com/bilibili/gengine/builder"
	"github.com/bilibili/gengine/context"
	"github.com/bilibili/gengine/engine"
	"github.com/pingcap/diag/checker/pkg/utils"
	"github.com/pingcap/diag/checker/proto"
	"testing"
)

const rule1 = `
rule "test"
begin
		if 0 == ToInt(config.GetValueByTagPath("log.file.max-days")){
			return -1
		}else{
			return 10
		}
end
`

func Test_WithReflect(t *testing.T){
	dataContext := context.NewDataContext()
	cfg := proto.NewPdConfigData()
	cfg.Log.File.MaxDays = 10
	dataContext.Add("config", cfg)
	dataContext.Add("ToInt", utils.ValueToInt)
	ruleBuilder := builder.NewRuleBuilder(dataContext)
	if err := ruleBuilder.BuildRuleFromString(rule1); err != nil {
		t.Fatal(err)
	}

	eng := engine.NewGengine()
	if err := eng.Execute(ruleBuilder, false); err != nil {
		t.Fatal(err)
	}
	result, _ := eng.GetRulesResultMap()
	t.Log(result)
}

func TestRuleCheck_Check(t *testing.T) {

}