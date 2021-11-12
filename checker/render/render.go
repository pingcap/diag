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
	"time"

	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/tiup/pkg/logger/log"
)

// bytes.buffer flush into checker_sampleid_timestamp.txt
type ResultWrapper struct {
	RuleSet map[string]*proto.Rule
	Data    *proto.SourceDataV2
}

func NewResultWrapper(rs map[string]*proto.Rule, data *proto.SourceDataV2) *ResultWrapper {
	return &ResultWrapper{
		RuleSet: rs,
		Data:    data,
	}
}

// data variable name, data variable value.
func (w *ResultWrapper) Output(checkresult map[string]proto.PrintTemplate) error {
	// todo@toto find rule check result
	// print OutputMetaData
	now := time.Now()
	sec := now.Unix()
	writer, err := NewCheckerWriter(fmt.Sprintf("checker-%s-%d.txt", w.Data.ClusterInfo.Session, sec))
	if err != nil {
		log.Errorf("create file failed, ", err.Error())
	}
	defer func() {
		writer.Flush()
		writer.Close()
	}()

	writer.WriteString(fmt.Sprintf("%s %s\n", w.Data.ClusterInfo.ClusterName, w.Data.ClusterInfo.BeginTime))
	writer.WriteString("# Cluster Information")
	writer.WriteString(fmt.Sprintln("ClusterId: ", w.Data.ClusterInfo.ClusterID))
	writer.WriteString(fmt.Sprintln("ClusterName: ", w.Data.ClusterInfo.ClusterName))
	writer.WriteString(fmt.Sprintln("ClusterVersoin: ", w.Data.TidbVersion))
	writer.WriteString("\n")

	writer.WriteString("# Sample Information")
	writer.WriteString(fmt.Sprintln("Sample ID: ", w.Data.ClusterInfo.Session))
	writer.WriteString(fmt.Sprintln("Sample Content: ", w.Data.ClusterInfo.Collectors))
	writer.WriteString("\n")
	for rulename, printer := range checkresult {
		rule, ok := w.RuleSet[rulename]
		if !ok {
			log.Errorf("unknown rule name for output ", rulename)
			continue
		}
		writer.WriteString("# Configuration Check Result\n")
		writer.WriteString(fmt.Sprintln("- RuleName: ", rulename))
		writer.WriteString(fmt.Sprintln("- RuleID: ", rule.ID))
		writer.WriteString(fmt.Sprintln("- Variation: ", rule.Variation))
		writer.WriteString(fmt.Sprintln("- Alerting Rule: ", rule.AlertingRule))
		writer.WriteString(fmt.Sprintln("- Check Result: "))
		printer.Print(writer)
		writer.WriteString("\n")
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
	if err := w.termWriter.Flush(); err != nil {
		return err
	}
	return nil
}

func NewCheckerWriter(filename string) (*CheckerWriter, error) {
	f, err := os.Create(filename) //
	if err != nil {
		log.Errorf("create file failed, ", err.Error())
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
