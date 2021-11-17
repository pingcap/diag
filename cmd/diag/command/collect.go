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
	"bufio"
	"os"
	"path"
	"time"

	"github.com/pingcap/diag/collector"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/tui"
	"github.com/pingcap/tiup/pkg/utils"
	"github.com/spf13/cobra"
)

func newCollectCmd() *cobra.Command {
	var collectAll bool
	var metricsConf string
	opt := collector.BaseOptions{
		SSH: &tui.SSHConnectionProps{
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

			if collectAll {
				cOpt.Include = collector.CollectAllSet
			} else if len(inc) > 0 {
				cOpt.Include = set.NewStringSet(inc...)
			}
			if len(ext) > 0 {
				cOpt.Exclude = set.NewStringSet(ext...)
			}
			opt.Cluster = args[0]

			if metricsConf != "" {
				f, err := os.Open(metricsConf)
				if err != nil {
					return err
				}
				defer f.Close()
				s := bufio.NewScanner(f)
				for s.Scan() {
					if len(s.Text()) > 0 {
						cOpt.MetricsFilter = append(cOpt.MetricsFilter, s.Text())
					}
				}
			}

			cOpt.Mode = collector.CollectModeTiUP
			cm.DisplayMode = gOpt.DisplayMode
			return cm.CollectClusterInfo(&opt, &cOpt, &gOpt, nil, nil)
		},
	}

	cmd.Flags().StringSliceVarP(&gOpt.Roles, "role", "R", nil, "Only collect data from specified roles")
	cmd.Flags().StringSliceVarP(&gOpt.Nodes, "node", "N", nil, "Only collect data from specified nodes")
	cmd.Flags().StringVarP(&opt.ScrapeBegin, "from", "f", time.Now().Add(time.Hour*-2).Format(time.RFC3339), "start timepoint when collecting timeseries data")
	cmd.Flags().StringVarP(&opt.ScrapeEnd, "to", "t", time.Now().Format(time.RFC3339), "stop timepoint when collecting timeseries data")
	cmd.Flags().BoolVar(&collectAll, "all", false, "Collect all data")
	cmd.Flags().StringSliceVar(&inc, "include", cOpt.Include.Slice(), "types of data to collect")
	cmd.Flags().StringSliceVar(&ext, "exclude", cOpt.Exclude.Slice(), "types of data not to collect")
	cmd.Flags().StringSliceVar(&cOpt.MetricsFilter, "metricsfilter", nil, "prefix of metrics to collect")
	cmd.Flags().StringVar(&metricsConf, "metricsconfig", "", "config file of metricsfilter")
	cmd.Flags().StringVarP(&cOpt.Dir, "output", "o", "", "output directory of collected data")
	cmd.Flags().IntVarP(&cOpt.Limit, "limit", "l", 10000, "Limits the used bandwidth, specified in Kbit/s")
	cmd.Flags().Uint64Var(&gOpt.APITimeout, "api-timeout", 10, "Timeout in seconds when querying PD APIs.")
	cmd.Flags().BoolVar(&cOpt.CompressMetrics, "compress-metrics", true, "Compress collected metrics data.")
	cmd.Flags().BoolVar(&cOpt.CompressScp, "compress-scp", true, "Compress when transfer config and logs.")
	cmd.Flags().BoolVar(&cOpt.ExitOnError, "exit-on-error", false, "Stop collecting and exit if an error occurs.")

	return cmd
}
