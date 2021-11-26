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

package main

import (
	"fmt"
	"os"

	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/scraper"
	"github.com/pingcap/diag/version"
	"github.com/spf13/cobra"
)

var (
	rootCmd *cobra.Command
	opt     = &scraper.Option{
		LogPaths:    make([]string, 0),
		ConfigPaths: make([]string, 0),
	}
)

func init() {
	rootCmd = &cobra.Command{
		Use:           "scraper",
		Short:         "Scrap logs and configs from a TiDB cluster",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.String(),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := Scrap(opt)
			if err != nil {
				return err
			}
			rb, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(rb))
			return nil
		},
	}

	rootCmd.Flags().StringSliceVar(&opt.LogPaths, "log", nil, "paths of log files to scrap")
	rootCmd.Flags().StringSliceVar(&opt.ConfigPaths, "config", nil, "paths of config files to scrap")
	rootCmd.Flags().StringVarP(&opt.Start, "from", "f", "", "start time of range to scrap, only apply to logs")
	rootCmd.Flags().StringVarP(&opt.End, "to", "t", "", "start time of range to scrap, only apply to logs")

	// time range is required, no default values are assumed
	cobra.MarkFlagRequired(rootCmd.Flags(), "from")
	cobra.MarkFlagRequired(rootCmd.Flags(), "to")
}

// Execute executes the root command
func Execute() {
	code := 0
	err := rootCmd.Execute()
	if err != nil {
		code = 1
	}

	if code != 0 {
		os.Exit(code)
	}
}

func main() {
	Execute()
}
