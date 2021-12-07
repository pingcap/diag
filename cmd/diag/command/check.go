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

package command

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/pingcap/diag/checker/config"
	"github.com/pingcap/diag/checker/engine"
	"github.com/pingcap/diag/checker/render"
	"github.com/pingcap/diag/checker/sourcedata"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func newCheckCmd() *cobra.Command {
	var datapath = ""
	var inc = []string{"config"}
	var logLevel = ""
	var outpath = ""
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check config collected from a TiDB cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if logLevel != "info" {
				l := zap.NewAtomicLevel()
				if l.UnmarshalText([]byte(logLevel)) == nil {
					zap.IncreaseLevel(l)
				}
			}
			zap.L().Debug("checker started")
			// TODO: integrate fetcher

			// todo: checker action id
			// todo: checker action time
			// todo: version
			var checkFlag sourcedata.CheckFlag
			for _, val := range inc {
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
			if len(outpath) == 0 {
				outpath = path.Join(datapath, fmt.Sprintf("report-%s", time.Now().Format("060102150405")))
			}
			fetch, err := sourcedata.NewFileFetcher(datapath,
				sourcedata.WithCheckFlag(checkFlag),
				sourcedata.WithOutputDir(outpath))
			if err != nil {
				log.Errorf("error fetching source data: %s", err)
				return err
			}
			ruleSpec, err := config.LoadBetaRuleSpec()
			if err != nil {
				log.Errorf("error loading rule specs: %s", err)
				return err
			}
			data, ruleSet, err := fetch.FetchData(ruleSpec)
			if err != nil {
				log.Errorf("error fetching data: %s", err)
				return err
			}
			include := strings.Join(inc, "-")
			render := render.NewResultWrapper(data, ruleSet, outpath, include)
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
				return wrapper.Start()
			})
			if err := errG.Wait(); err != nil {
				log.Errorf("check meet error: %s", err)
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&datapath, "datapath", "./data", "path to collected data")
	cmd.Flags().StringVar(&logLevel, "loglevel", "info", "log level, supported value is debug, info")
	cmd.Flags().StringVarP(&outpath, "output", "o", "", "dir to save check report. report will be saved in datapath if not set")
	cmd.Flags().StringSliceVar(&inc, "include", inc, "types of data to check, supported value is config, performance, default_config")
	return cmd
}
