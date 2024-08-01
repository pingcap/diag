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

package checker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pingcap/diag/checker/config"
	"github.com/pingcap/diag/checker/engine"
	"github.com/pingcap/diag/checker/render"
	"github.com/pingcap/diag/checker/sourcedata"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"golang.org/x/sync/errgroup"
)

// Options are configs for checker
type Options struct {
	DataPath string
	Inc      []string
	OutPath  string
}

// NewOptions creates a default Options
func NewOptions() *Options {
	return &Options{
		Inc: []string{"config"},
	}
}

// RunCheck do the checks
func (opt *Options) RunChecker(ctx context.Context) error {
	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)
	// TODO: integrate fetcher

	// todo: checker action id
	// todo: checker action time
	// todo: version
	var checkFlag sourcedata.CheckFlag
	for _, val := range opt.Inc {
		if val == "config" {
			checkFlag |= sourcedata.ConfigFlag
		}
		if val == "performance" {
			checkFlag |= sourcedata.PerformanceFlag
		}
		if val == "default_config" {
			checkFlag |= sourcedata.DefaultConfigFlag
		}
	}
	// if output is not defined, use an auto generated one.
	if len(opt.OutPath) == 0 {
		opt.OutPath = filepath.Join(opt.DataPath, fmt.Sprintf("report-%s", time.Now().Format("060102150405")))
	}
	fetch, err := sourcedata.NewFileFetcher(opt.DataPath,
		sourcedata.WithCheckFlag(checkFlag),
		sourcedata.WithOutputDir(opt.OutPath))
	if err != nil {
		logger.Errorf("error fetching source data: %s", err)
		return err
	}
	ruleSpec, err := config.LoadBetaRuleSpec()
	if err != nil {
		logger.Errorf("error loading rule specs: %s", err)
		return err
	}
	data, ruleSet, err := fetch.FetchData(ruleSpec)
	if err != nil {
		logger.Errorf("error fetching data: %s", err)
		return err
	}
	inc := strings.Join(opt.Inc, "-")
	render := render.NewResultWrapper(data, ruleSet, opt.OutPath, inc)
	wrapper := engine.NewWrapper(data, ruleSet, render)
	// checkline.Init()
	// pipe := checkline.GetResultChan()
	// screenRender := render.NewScreenRender(pipe)
	errG, _ := errgroup.WithContext(context.Background())

	// todo receive context
	// cluster id
	// cluster name
	// cluster version
	// cluster Time
	// Uptime

	// SampleId
	// Sampling Date
	// Sample Content
	// - configuration: tidb, tikv, pd
	// - performance: tidb dashboard, prometheus
	// - Rule ID: <String> // e.g. 12343234334，全局唯一
	// - Variation: <String> // e.g. tidb.file.max_days

	// fetch unique num +1

	errG.Go(func() error {
		return wrapper.Start(ctx)
	})
	if err := errG.Wait(); err != nil {
		logger.Errorf("check meet error: %s", err)
		return err
	}
	return nil
}
