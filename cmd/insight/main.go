// Copyright 2018 PingCAP, Inc.
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
	"log"

	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/collector/sysinfo"
	"github.com/pingcap/diag/version"
	"github.com/spf13/cobra"
)

var (
	insightCmd *cobra.Command
	opts       sysinfo.Options
)

func init() {
	insightCmd = &cobra.Command{
		Use:     "insight",
		Short:   "A system information collector.",
		Version: version.String(),
		RunE: func(cmd *cobra.Command, args []string) error {
			var info sysinfo.InsightInfo
			info.GetInfo(opts)

			data, err := json.MarshalIndent(&info, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(data))
			return nil
		},
	}

	insightCmd.Flags().BoolVar(&opts.Dmesg, "dmesg", false, "Also collect kernel logs.")
	insightCmd.Flags().BoolVar(&opts.Proc, "proc", false, "Also collect process list.")
	insightCmd.Flags().BoolVar(&opts.Syscfg, "syscfg", false, "Also collect system configs.")
}

func main() {
	if err := insightCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
