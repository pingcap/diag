// Copyright 2022 PingCAP, Inc.
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
	"fmt"
	"os"

	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/spf13/cobra"
)

const RegionInfo = `Clinic Server provides the following two regions to store your diagnostic data:
[CN] region: Data stored in China Mainland, domain name : https://clinic.pingcap.com.cn
[US] region: Data stored in USA ,domain name : https://clinic.pingcap.com`

func newConfigCmd() *cobra.Command {
	var unset, show bool
	cmd := &cobra.Command{
		Use:   "config <key> [value] [--unset]",
		Short: "set an individual value in diag configuration file",
		Long: `set an individual value in diag configuration file, like
  "diag config clinic.token xxxxxxxxxx"
if not specify key nor value, an interactive interface will be used to set necessary configuration`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if show {
				confStr, err := os.ReadFile(spec.ProfilePath("diag.toml"))
				if err != nil {
					return err
				}
				fmt.Println(string(confStr))
			} else if unset {
				if len(args) != 1 {
					return cmd.Help()
				}
				err := diagConfig.Unset(args[0])
				if err != nil {
					return err
				}
			} else {
				switch len(args) {
				case 0:
					diagConfig.InteractiveSet()
				case 2:
					err := diagConfig.Set(args[0], args[1])
					if err != nil {
						return err
					}
				default:
					return cmd.Help()
				}
			}
			return diagConfig.Save()
		},
	}

	cmd.PersistentFlags().BoolVar(&unset, "unset", false, "unset diag configuration")
	cmd.PersistentFlags().BoolVar(&show, "show", false, "show diag configuration")

	return cmd
}
