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
	"github.com/BurntSushi/toml"
	"testing"
)

var ruleConfig = `
[[rule]]
id = 7
name = "log-level"
description = "日志等级。可选值：\"trace\"，\"debug\"，\"info\"，\"warning\"，\"error\"，\"critical\"。默认值：\"info\""
variation =  "log-level"
execute_rule = """
rule "log-level" "log level of tikv"  salience 6
begin
    if ToString(config.GetValueByTagPath("log-level")) == "debug" || ToString(config.GetValueByTagPath("log-level")) == "trace" {
        return false
    } else {
        return true
    }
end
"""
name_struct = "TikvConfig"
expect_res = ""   #
warn_level = "info"
version = ""
`

func TestRuleSpec(t *testing.T) {
	rules := RuleSpec{}
	if _, err := toml.Decode(ruleConfig, &rules); err != nil {
		t.Fatal(err)
	}
	if len(rules.Rule) == 0 {
		t.Fatal("wrong toml decode result")
	}
	t.Logf("%+v", rules)
}

func TestLoadBetaRuleSpec(t *testing.T) {
	_, err := LoadBetaRuleSpec()
	if err != nil {
		t.Error(err)
	}
}
