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
	"bufio"
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/tiup/pkg/logger/log"
)

// bytes.buffer flush into checker_sampleid_timestamp.txt
type ResultWrapper struct {
	RuleSet   map[string]*proto.Rule
	Data      *proto.SourceDataV2
	storePath string
	include   string
}

func NewResultWrapper(data *proto.SourceDataV2, rs map[string]*proto.Rule, sp string, inc string) *ResultWrapper {
	return &ResultWrapper{
		RuleSet:   rs,
		Data:      data,
		storePath: sp,
		include:   inc,
	}
}

func (w *ResultWrapper) GroupByType() (map[string][]*proto.Rule, []string) {
	ruleMapping := make(map[string][]*proto.Rule)
	keys := make([]string, 0)
	for _, rule := range w.RuleSet {
		ruleslice, ok := ruleMapping[rule.CheckType]
		if ok {
			ruleslice = append(ruleslice, rule)
		} else {
			ruleslice = []*proto.Rule{rule}
			keys = append(keys, rule.CheckType)
		}
		ruleMapping[rule.CheckType] = ruleslice
	}
	for _, rs := range ruleMapping {
		sort.Slice(rs, func(i, j int) bool {
			return rs[i].ID < rs[j].ID
		})
	}
	sort.Slice(keys, func(i, j int) bool {
		left := keys[i]
		right := keys[j]
		return proto.CheckTypeOrder[left] < proto.CheckTypeOrder[right]
	})
	return ruleMapping, keys
}

// data variable name, data variable value.
func (w *ResultWrapper) Output(checkresult map[string]proto.PrintTemplate) error {
	// todo@toto find rule check result
	// print OutputMetaData
	writer, err := NewCheckerWriter(w.storePath, fmt.Sprintf("%s-%s.txt", "checker", w.include))
	if err != nil {
		log.Errorf("create file failed %+v", err.Error())
		return err
	}
	defer func() {
		writer.Flush()
		writer.Close()
		fmt.Printf("Result report is saved at %s\n", w.storePath)
	}()
	writer.WriteString("# Check Result Report\n")
	writer.WriteString(fmt.Sprintf("%s %s\n\n", w.Data.ClusterInfo.ClusterName, w.Data.ClusterInfo.BeginTime))

	writer.WriteString("## 1. Cluster Information\n")
	writer.WriteString(fmt.Sprintln("- Cluster ID: ", w.Data.ClusterInfo.ClusterID))
	writer.WriteString(fmt.Sprintln("- Cluster Name: ", w.Data.ClusterInfo.ClusterName))
	writer.WriteString(fmt.Sprintln("- Cluster Version: ", w.Data.TidbVersion))
	writer.WriteString("\n")

	writer.WriteString("## 2. Sample Information\n")
	writer.WriteString(fmt.Sprintln("- Sample ID: ", w.Data.ClusterInfo.Session))
	writer.WriteString(fmt.Sprintln("- Sampling Date: ", w.Data.ClusterInfo.BeginTime))
	writer.WriteString(fmt.Sprintln("- Sample Content:: ", w.Data.ClusterInfo.Collectors))
	writer.WriteString("\n")

	writer.WriteString("## 3. Check Result\n")

	typeRules, keys := w.GroupByType()
	for _, ruleType := range keys {
		rules := typeRules[ruleType]
		if ruleType == proto.ConfigType {
			writer.WriteString("### Configuration\n")
		} else if ruleType == proto.PerformanceType {
			writer.WriteString("### SQL Performance\n")
		} else if ruleType == proto.DefaultConfigType {
			writer.WriteString("### Default Configuration\n")
		}
		for _, rule := range rules {
			printer, ok := checkresult[rule.Name]
			if !ok {
				log.Errorf("No such rule result")
				continue
			}
			writer.WriteString(fmt.Sprintln("#### Rule Name: ", rule.Name))
			writer.WriteString(fmt.Sprintln("- RuleID: ", rule.ID))
			writer.WriteString(fmt.Sprintln("- Variation: ", rule.Variation))
			if len(rule.AlertingRule) > 0 {
				writer.WriteString(fmt.Sprintln("- Alerting Rule: ", rule.AlertingRule))
			}
			if len(rule.ExpectRes) > 0 {
				writer.WriteString(fmt.Sprintln("- For more information, please visit: ", rule.ExpectRes))
			}
			writer.WriteString(fmt.Sprintln("- Check Result: "))
			printer.Print(writer)
			writer.WriteString("\n")
		}
	}
	return nil
}

type CheckerWriter struct {
	termWriter *bufio.Writer
	fileWriter *bufio.Writer
	f          *os.File
}

func (w *CheckerWriter) Flush() error {
	if err := w.fileWriter.Flush(); err != nil {
		return err
	}
	return w.termWriter.Flush()
}

func NewCheckerWriter(dirPath string, filename string) (*CheckerWriter, error) {
	if err := os.MkdirAll(dirPath, 0777); err != nil {
		log.Errorf("create path failed, %+v", err.Error())
		return nil, err
	}
	f, err := os.Create(path.Join(dirPath, filename))
	if err != nil {
		log.Errorf("create file failed, %+v", err.Error())
		return nil, err
	}
	termwriter := bufio.NewWriter(f)
	return &CheckerWriter{
		fileWriter: termwriter,
		termWriter: bufio.NewWriter(os.Stdout),
		f:          f}, nil

}

func (w *CheckerWriter) WriteString(info string) {
	w.fileWriter.WriteString(info)
	w.termWriter.WriteString(info)
}

func (w *CheckerWriter) Write(p []byte) (nn int, err error) {
	nn, err = w.fileWriter.Write(p)
	if err != nil {
		return 0, err
	}
	_, err = w.termWriter.Write(p)
	if err != nil {
		return 0, err
	}
	return nn, err
}

func (w *CheckerWriter) Close() {
	w.f.Close()
}
