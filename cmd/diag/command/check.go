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

	"github.com/pingcap/diag/checker/config"
	"github.com/pingcap/diag/checker/engine"
	"github.com/pingcap/diag/checker/sourcedata"
	"github.com/pingcap/log"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func newCheckCmd() *cobra.Command {
	var datapath = ""
	var inc = []string{"config"}
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check config collected from a TiDB cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Debug("start checker")
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
			}
			fetch, err := sourcedata.NewFileFetcher(datapath, sourcedata.WithCheckFlag(checkFlag))
			if err != nil {
				log.Error(err.Error())
				return err
			}
			ruleSpec, err := config.LoadBetaRuleSpec()
			if err != nil {
				log.Error(err.Error())
				return err
			}
			data, ruleSet, err := fetch.FetchData(ruleSpec)
			if err != nil {
				log.Error(err.Error())
				return err
			}
			wrapper := engine.NewWrapper(data, ruleSet)
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
				log.Error("check meet error: %+v", zap.Error(err))
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&datapath, "datapath", "./data", "path to collected data")
	cmd.Flags().StringSliceVar(&inc, "include", inc, "types of data to check, supported value is config, performance")
	return cmd
}
