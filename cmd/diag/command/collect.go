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
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pingcap/diag/collector"
	"github.com/pingcap/diag/pkg/telemetry"
	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/tui"
	tiuputils "github.com/pingcap/tiup/pkg/utils"
	"github.com/spf13/cobra"
)

func newCollectCmd() *cobra.Command {
	var collectAll bool
	var metricsConf string
	opt := collector.BaseOptions{
		SSH: &tui.SSHConnectionProps{
			IdentityFile: path.Join(tiuputils.UserHome(), ".ssh", "id_rsa"),
		},
	}
	cOpt := collector.CollectOptions{
		Include: collector.CollectDefaultSet,
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
			cOpt.RawRequest = strings.Join(os.Args[1:], " ")

			log.SetDisplayModeFromString(gOpt.DisplayMode)
			spec.Initialize("cluster")
			tidbSpec := spec.GetSpecManager()
			cm := collector.NewManager("tidb", tidbSpec, log)

			// natvie ssh has it's own logic to find the default identity_file
			if gOpt.SSHType == executor.SSHTypeSystem && !tiuputils.IsFlagSetByUser(cmd.Flags(), "identity_file") {
				opt.SSH.IdentityFile = ""
			}

			if collectAll {
				cOpt.Include.Join(collector.CollectAdditionSet)
			} else if len(inc) > 0 {
				cOpt.Include = set.NewStringSet(inc...)
			}
			if len(ext) > 0 {
				cOpt.Exclude = set.NewStringSet(ext...)
			}
			opt.Cluster = args[0]
			clsID := scrubClusterName(opt.Cluster)
			teleCommand = append(teleCommand, clsID)

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

			if cOpt.Limit == -1 {
				switch gOpt.SSHType {
				case "system":
					cOpt.Limit = 100000
				default:
					cOpt.Limit = 10000
				}
			}
			cOpt.Mode = collector.CollectModeTiUP

			if reportEnabled {
				teleReport.CommandInfo = &telemetry.CollectInfo{
					ID:         clsID,
					Mode:       cOpt.Mode,
					ArgYes:     skipConfirm,
					ArgLimit:   cOpt.Limit,
					ArgInclude: cOpt.Include.Slice(),
					ArgExclude: cOpt.Exclude.Slice(),
				}
			}

			_, err := cm.CollectClusterInfo(&opt, &cOpt, &gOpt, nil, nil, skipConfirm)
			// time is validated and updated during the collecting process
			if reportEnabled {
				st, errs := utils.ParseTime(opt.ScrapeBegin)
				et, erre := utils.ParseTime(opt.ScrapeEnd)
				if errs == nil && erre == nil {
					teleReport.CommandInfo.(*telemetry.CollectInfo).
						TimeSpan = int64(et.Sub(st))
				}

				if size, err := utils.DirSize(cOpt.Dir); err == nil {
					teleReport.CommandInfo.(*telemetry.CollectInfo).
						DataSize = size
				}
			}
			return err
		},
	}

	cmd.Flags().StringVar(&cOpt.ProfileName, "profile", "", "File name of a pre-defined collecting profile")
	cmd.Flags().StringSliceVarP(&gOpt.Roles, "role", "R", nil, "Only collect data from specified roles")
	cmd.Flags().StringSliceVarP(&gOpt.Nodes, "node", "N", nil, "Only collect data from specified nodes")
	cmd.Flags().StringVarP(&opt.ScrapeBegin, "from", "f", time.Now().Add(time.Hour*-2).Format(time.RFC3339), "start timepoint when collecting timeseries data")
	cmd.Flags().StringVarP(&opt.ScrapeEnd, "to", "t", time.Now().Format(time.RFC3339), "stop timepoint when collecting timeseries data")
	cmd.Flags().BoolVar(&collectAll, "all", false, "Collect all data")
	cmd.Flags().StringSliceVar(&inc, "include", nil, fmt.Sprintf("types of data to collect: default[%s] non-default[%s]", strings.Join(collector.CollectDefaultSet.Slice(), ","), strings.Join(collector.CollectAdditionSet.Slice(), ",")))
	cmd.Flags().StringSliceVar(&ext, "exclude", nil, "types of data not to collect")
	cmd.Flags().StringSliceVar(&cOpt.MetricsFilter, "metricsfilter", nil, "prefix of metrics to collect")
	cmd.Flags().StringVar(&metricsConf, "metricsconfig", "", "config file of metricsfilter")
	cmd.Flags().StringVarP(&cOpt.Dir, "output", "o", "", "output directory of collected data")
	cmd.Flags().IntVarP(&cOpt.Limit, "limit", "l", -1, "Limits the used bandwidth, specified in Kbit/s")
	cmd.Flags().IntVar(&cOpt.PerfDuration, "perf-duration", 30, "Duration of the collection of profile information in seconds")
	cmd.Flags().Uint64Var(&gOpt.APITimeout, "api-timeout", 10, "Timeout in seconds when querying PD APIs.")
	cmd.Flags().BoolVar(&cOpt.CompressScp, "compress-scp", true, "Compress when transfer config and logs.Only works with system ssh")
	cmd.Flags().BoolVar(&cOpt.ExitOnError, "exit-on-error", false, "Stop collecting and exit if an error occurs.")

	return cmd
}
