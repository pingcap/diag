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

	"github.com/pingcap/diag/checker"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newCheckCmd() *cobra.Command {
	var logLevel string
	opt := checker.NewOptions()
	cmd := &cobra.Command{
		Use:   "check <collected-datadir>",
		Short: "Check config collected from a TiDB cluster",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}
			opt.DataPath = args[0]
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if logLevel != "info" {
				l := zap.NewAtomicLevel()
				if l.UnmarshalText([]byte(logLevel)) == nil {
					zap.IncreaseLevel(l)
				}
			}
			zap.L().Debug("checker started")

			return opt.RunChecker(context.WithValue(context.Background(), logprinter.ContextKeyLogger, log))
		},
	}

	cmd.Flags().StringVar(&logLevel, "loglevel", "info", "log level, supported value is debug, info")
	cmd.Flags().StringVarP(&opt.OutPath, "output", "o", "", "dir to save check report. report will be saved in datapath if not set")
	cmd.Flags().StringSliceVar(&opt.Inc, "include", opt.Inc, "types of data to check, supported value is config, performance, default_config")
	return cmd
}
