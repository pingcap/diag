// Copyright 2020 PingCAP, Inc.
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
	"path"

	"github.com/pingcap/tidb-foresight/collector"
	"github.com/pingcap/tiup/pkg/cliutil"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/utils"
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	opt := collector.BaseOptions{
		SSH: &cliutil.SSHConnectionProps{
			IdentityFile: path.Join(utils.UserHome(), ".ssh", "id_rsa"),
		},
	}
	cOpt := collector.CollectOptions{
		Include: set.NewStringSet( // collect all types by default
			collector.CollectTypeSystem,
			collector.CollectTypeMonitor,
			collector.CollectTypeLog,
			collector.CollectTypeConfig,
		),
		Exclude: set.NewStringSet(),
	}
	inc := make([]string, 0)
	ext := make([]string, 0)
	cmd := &cobra.Command{
		Use:   "collect <cluster-name>",
		Short: "Collect information and metrics from the cluster.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}
			// natvie ssh has it's own logic to find the default identity_file
			if gOpt.SSHType == executor.SSHTypeSystem && !utils.IsFlagSetByUser(cmd.Flags(), "identity_file") {
				opt.SSH.IdentityFile = ""
			}

			if len(inc) > 0 {
				cOpt.Include = set.NewStringSet(inc...)
			}
			if len(ext) > 0 {
				cOpt.Exclude = set.NewStringSet(ext...)
			}
			return cm.CollectClusterInfo(args[0], &opt, &cOpt, &gOpt)
		},
	}

	cmd.Flags().StringVarP(&opt.User, "user", "u", utils.CurrentUser(), "The user name to login via SSH. The user must has root (or sudo) privilege.")
	cmd.Flags().StringVarP(&opt.SSH.IdentityFile, "identity_file", "i", opt.SSH.IdentityFile, "The path of the SSH identity file. If specified, public key authentication will be used.")
	cmd.Flags().BoolVarP(&opt.UsePassword, "password", "p", false, "Use password of target hosts. If specified, password authentication will be used.")
	cmd.Flags().StringSliceVarP(&gOpt.Roles, "role", "R", nil, "Only check specified roles")
	cmd.Flags().StringSliceVarP(&gOpt.Nodes, "node", "N", nil, "Only check specified nodes")
	cmd.Flags().Uint64Var(&gOpt.APITimeout, "api-timeout", 10, "Timeout in seconds when querying PD APIs.")
	cmd.Flags().StringVar(&opt.ScrapeBegin, "begin", "", "start timepoint when collecting timeseries data")
	cmd.Flags().StringVar(&opt.ScrapeEnd, "end", "", "stop timepoint when collecting timeseries data")
	cmd.Flags().StringSliceVar(&inc, "include", cOpt.Include.Slice(), "types of data to collect")
	cmd.Flags().StringSliceVar(&ext, "exclude", cOpt.Exclude.Slice(), "types of data not to collect")

	return cmd
}
