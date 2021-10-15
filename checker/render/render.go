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
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/lensesio/tableprinter"
	"github.com/pingcap/diag/checker/proto"
	"github.com/sirupsen/logrus"
)

type Render interface {
	Output(data map[string]interface{}) error
}

type ScreenRender struct {
	// TODO: remove logger
	logger logrus.StdLogger
	source <-chan map[string]proto.RuleResult
	//	printer *tableprinter.Printer
}

func NewScreenRender(source <-chan map[string]proto.RuleResult) *ScreenRender {
	// printer := tableprinter.New(os.Stdout)
	// header := tableprinter.StructParser.ParseHeaders(reflect.ValueOf(&proto.DeployResult{}))
	// printer.Render(header, nil, nil, false)
	return &ScreenRender{
		logger: logrus.StandardLogger(),
		source: source,
		//printer: printer,
	}
}

// change to channel
func (r *ScreenRender) Output(ctx context.Context) error {
	for {
		select {
		case data, ok := <-r.source:
			if !ok {
				return nil
			}
			for _, result := range data {
				fmt.Println("# Configuration Check Result")
				fmt.Println("- RuleName: ", result.RuleName)
				fmt.Println("- RuleID: ", result.RuleID)
				fmt.Println("- Variation: ", result.Variation)
				fmt.Println("- Alerting Rule: ", result.AlertingRule)
				fmt.Println("- Check Result: ")
				printer := tableprinter.New(os.Stdout)
				// _ := tableprinter.StructParser.ParseHeaders(reflect.ValueOf(&proto.DeployResult{}))
				// printer.Render(header, nil, nil, false)
				for _, rr := range result.DeployResults {
					row, nums := tableprinter.StructParser.ParseRow(reflect.ValueOf(rr))
					printer.RenderRow(row, nums)
				}
				fmt.Print("\n")
			}
		case <-ctx.Done():
			return nil
		}
	}

}
