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

package render

import (
	"fmt"

	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/tiup/pkg/logger/log"
)

type ResultWrapper struct {
	RuleSet map[string]*proto.Rule
	// outputMetaData proto.OutputMetaData
}

func NewResultWrapper(rs map[string]*proto.Rule) *ResultWrapper {
	return &ResultWrapper{
		RuleSet: rs,
	}
}

// data variable name, data variable value.
func (w *ResultWrapper) Output(checkresult map[string]proto.PrintTemplate) error {
	// todo@toto find rule check result
	// print OutputMetaData
	for rulename, printer := range checkresult {
		rule, ok := w.RuleSet[rulename]
		if !ok {
			log.Errorf("unknown rule name for output ", rulename)
			continue
		}
		fmt.Println("# Configuration Check Result")
		fmt.Println("- RuleName: ", rulename)
		fmt.Println("- RuleID: ", rule.ID)
		fmt.Println("- Variation: ", rule.Variation)
		fmt.Println("- Alerting Rule: ", rule.AlertingRule)
		fmt.Println("- Check Result: ")
		printer.Print() // change print
		fmt.Print("\n")
	}
	return nil
}
