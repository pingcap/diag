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
	"context"
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/pingcap/diag/checker/proto"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
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
func (w *ResultWrapper) Output(ctx context.Context, checkresult map[string]proto.PrintTemplate) error {
	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)
	// todo@toto find rule check result
	// print OutputMetaData
	defer func() {
		logger.Infof("Result report and record are saved at %s", w.storePath)
	}()
	if err := w.OutputSummary(logger, checkresult); err != nil {
		return err
	}
	return w.SaveDetail(checkresult)
}

func (w *ResultWrapper) OutputSummary(logger *logprinter.Logger, checkresult map[string]proto.PrintTemplate) error {
	// todo@toto find rule check result
	// print OutputMetaData
	writer, err := NewCheckerWriter(w.storePath, "check-report.txt")
	if err != nil {
		logprinter.Errorf("create file failed %+v", err.Error())
		return err
	}
	defer func() {
		writer.Flush()
		writer.Close()
	}()
	writer.WriteString(logger, "# Check Result Report")
	writer.WriteString(logger, fmt.Sprintf("%s %s", w.Data.ClusterInfo.ClusterName, w.Data.ClusterInfo.BeginTime))

	writer.WriteString(logger, "\n## 1. Cluster Information")
	writer.WriteString(logger, fmt.Sprint("- Cluster ID: ", w.Data.ClusterInfo.ClusterID))
	writer.WriteString(logger, fmt.Sprint("- Cluster Name: ", w.Data.ClusterInfo.ClusterName))
	writer.WriteString(logger, fmt.Sprint("- Cluster Version: ", w.Data.TidbVersion))

	writer.WriteString(logger, "\n## 2. Sample Information")
	writer.WriteString(logger, fmt.Sprint("- Sample ID: ", w.Data.ClusterInfo.Session))
	writer.WriteString(logger, fmt.Sprint("- Sampling Date: ", w.Data.ClusterInfo.BeginTime))
	writer.WriteString(logger, fmt.Sprint("- Sample Content:: ", w.Data.ClusterInfo.Collectors))

	total, abnormalTotalCnt, abnormalConfigCnt, abnormalDefaultConfigCnt := 0, 0, 0, 0
	typeRules, keys := w.GroupByType()
	for _, ruleType := range keys {
		rules := typeRules[ruleType]
		total += len(rules)
		for _, rule := range rules {
			printer, ok := checkresult[rule.Name]
			if !ok {
				logprinter.Errorf("No such rule result")
				continue
			}
			if !printer.ResultAbnormal() {
				continue
			}
			abnormalTotalCnt++
			if ruleType == proto.ConfigType {
				abnormalConfigCnt++
			} else if ruleType == proto.DefaultConfigType {
				abnormalDefaultConfigCnt++
			}
		}
	}
	writer.WriteString(logger, "\n## 3. Main results and abnormalities")
	writer.WriteString(logger, fmt.Sprintf("In this inspection, %v rules were executed.\nThe results of **%v** rules were abnormal and needed to be further discussed with support team.\nThe following is the details of the abnormalities.",
		total, abnormalTotalCnt))

	for _, ruleType := range keys {
		rules := typeRules[ruleType]
		if ruleType == proto.ConfigType {
			writer.WriteString(logger, "\n### Configuration Summary")
			writer.WriteString(logger, fmt.Sprintf("The configuration rules are all derived from PingCAP’s OnCall Service.\nIf the results of the configuration rules are found to be abnormal, they may cause the cluster to fail.\nThere were **%v** abnormal results.",
				abnormalConfigCnt))

		} else if ruleType == proto.PerformanceType {
			continue
		} else if ruleType == proto.DefaultConfigType {
			writer.WriteString(logger, "\n### Default Configuration Summary")
			writer.WriteString(logger, fmt.Sprintf("The default configuration rules can find out which configurations are inconsistent with the default values.\nIf configurations were modified inadvertently, you can change they back to the default value based on this feedback.\nThere were **%v** abnormal results.",
				abnormalDefaultConfigCnt))
		}
		for _, rule := range rules {
			printer, ok := checkresult[rule.Name]
			if !ok {
				logprinter.Errorf("No such rule result")
				continue
			}
			if !printer.ResultAbnormal() {
				continue
			}
			writer.WriteString(logger, fmt.Sprint("\n#### Rule Name: ", rule.Name))
			writer.WriteString(logger, fmt.Sprint("- RuleID: ", rule.ID))
			writer.WriteString(logger, fmt.Sprint("- Variation: ", rule.Variation))
			if len(rule.AlertingRule) > 0 {
				writer.WriteString(logger, fmt.Sprint("- Alerting Rule: ", rule.AlertingRule))
			}
			if len(rule.ExpectRes) > 0 {
				writer.WriteString(logger, fmt.Sprint("- For more information, please visit: ", rule.ExpectRes))
			}
			writer.WriteString(logger, "- Check Result: ")
			loggerWrapper := writer.WrapLogger(logger)
			printer.Print(loggerWrapper)
			if err := loggerWrapper.Flush(); err != nil {
				return err
			}
		}
	}

	return nil
}

// SaveDetail write the content to file without print them.
func (w *ResultWrapper) SaveDetail(checkresult map[string]proto.PrintTemplate) error {
	// todo@toto find rule check result
	// print OutputMetaData
	writer, err := NewCheckerWriter(w.storePath, "detailed-check-record.txt")
	if err != nil {
		logprinter.Errorf("create file failed %+v", err.Error())
		return err
	}
	defer func() {
		writer.Flush()
		writer.Close()
	}()
	writer.SaveString("## Check Result Log")

	typeRules, keys := w.GroupByType()
	for _, ruleType := range keys {
		rules := typeRules[ruleType]
		if ruleType == proto.ConfigType {
			writer.SaveString("\n### Configuration")
		} else if ruleType == proto.PerformanceType {
			writer.SaveString("\n### SQL Performance")
		} else if ruleType == proto.DefaultConfigType {
			writer.SaveString("\n### Default Configuration")
		}
		for _, rule := range rules {
			printer, ok := checkresult[rule.Name]
			if !ok {
				logprinter.Errorf("No such rule result")
				continue
			}
			writer.SaveString(fmt.Sprint("\n#### Rule Name: ", rule.Name))
			writer.SaveString(fmt.Sprint("- RuleID: ", rule.ID))
			writer.SaveString(fmt.Sprint("- Variation: ", rule.Variation))
			if len(rule.AlertingRule) > 0 {
				writer.SaveString(fmt.Sprint("- Alerting Rule: ", rule.AlertingRule))
			}
			if len(rule.ExpectRes) > 0 {
				writer.SaveString(fmt.Sprint("- For more information, please visit: ", rule.ExpectRes))
			}
			writer.SaveString("- Check Result: ")
			printer.Print(writer.fileWriter)
		}
	}
	return nil
}

type CheckerWriter struct {
	fileWriter *bufio.Writer
	f          *os.File
}

func (w *CheckerWriter) Flush() error {
	return w.fileWriter.Flush()
}

func NewCheckerWriter(dirPath string, filename string) (*CheckerWriter, error) {
	if err := os.MkdirAll(dirPath, 0777); err != nil {
		logprinter.Errorf("create path failed, %+v", err.Error())
		return nil, err
	}
	f, err := os.Create(path.Join(dirPath, filename))
	if err != nil {
		logprinter.Errorf("create file failed, %+v", err.Error())
		return nil, err
	}
	return &CheckerWriter{
		fileWriter: bufio.NewWriter(f),
		f:          f}, nil
}

// WriteString write content to a file and the given logger TODO handle error
func (w *CheckerWriter) WriteString(logger *logprinter.Logger, info string) {
	_, _ = w.fileWriter.WriteString(info + "\n")
	logger.Infof(info)
}

// SaveString only write content to a file TODO handle error
func (w *CheckerWriter) SaveString(info string) {
	_, _ = w.fileWriter.WriteString(info + "\n")
}

// PrintString only write content to logger TODO handle error
func (w *CheckerWriter) PrintString(logger *logprinter.Logger, info string) {
	logger.Infof(info)
}

func (w *CheckerWriter) Write(logger *logprinter.Logger, p []byte) (nn int, err error) {
	nn, err = w.fileWriter.Write(p)
	if err != nil {
		return 0, err
	}
	logger.Infof("%s", p)
	return nn, err
}

type LoggerWriter struct {
	*logprinter.Logger
}

func (w *LoggerWriter) Write(p []byte) (nn int, err error) {
	s := string(p)
	w.Logger.Infof(s)
	return len(p), nil
}

type WriterWrapper struct {
	termWriter *bufio.Writer
	fileWriter *bufio.Writer
}

func (w *WriterWrapper) Flush() error {
	if err := w.fileWriter.Flush(); err != nil {
		return err
	}
	if err := w.termWriter.Flush(); err != nil {
		return err
	}
	return nil
}

func (w *WriterWrapper) Write(p []byte) (nn int, err error) {
	nn, err = w.fileWriter.Write(p)
	if err != nil {
		return 0, err
	}
	if _, err := w.termWriter.Write(p); err != nil {
		return 0, err
	}
	return nn, nil
}

func (w *CheckerWriter) WrapLogger(logger *logprinter.Logger) *WriterWrapper {
	bufWriter := bufio.NewWriter(&LoggerWriter{logger})
	return &WriterWrapper{
		termWriter: bufWriter,
		fileWriter: w.fileWriter,
	}
}

func (w *CheckerWriter) Close() {
	w.f.Close()
}
