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
	"reflect"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pingcap/diag/collector"
	"github.com/pingcap/diag/pkg/telemetry"
	"github.com/pingcap/diag/pkg/utils"
	dmspec "github.com/pingcap/tiup/components/dm/spec"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/tui"
	tiuputils "github.com/pingcap/tiup/pkg/utils"
	"github.com/spf13/cobra"
)

func newCollectDMCmd() *cobra.Command {
	var collectAll bool
	var metricsConf string
	opt := collector.BaseOptions{
		SSH: &tui.SSHConnectionProps{
			IdentityFile: path.Join(tiuputils.UserHome(), ".ssh", "id_rsa"),
		},
	}
	var cOpt collector.CollectOptions
	inc := make([]string, 0)
	ext := make([]string, 0)

	cmd := &cobra.Command{
		Use:   "collectdm <cluster-name>",
		Short: "Collect information and metrics from the dm cluster.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}
			cOpt.DiagMode = collector.DiagModeCmd
			cOpt.RawRequest = strings.Join(os.Args[1:], " ")

			log.SetDisplayModeFromString(gOpt.DisplayMode)
			spec.Initialize("dm")
			dmSpec := dmspec.GetSpecManager()
			cm := collector.NewManager("dm", dmSpec, log)

			// natvie ssh has it's own logic to find the default identity_file
			if gOpt.SSHType == executor.SSHTypeSystem && !tiuputils.IsFlagSetByUser(cmd.Flags(), "identity_file") {
				opt.SSH.IdentityFile = ""
			}

			if collectAll {
				utils.RecursiveSetBoolValue(reflect.ValueOf(&cOpt.Collectors).Elem(), true)
			} else {
				var err error
				cOpt.Collectors, err = collector.ParseCollectTree(inc, ext)
				if err != nil {
					return err
				}
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

			if cOpt.CompressMetrics == false {
				log.Warnf(color.YellowString("Uncompressed metrics may not be handled correctly by Clinic, use it only when you really need it"))
			}

			cOpt.Mode = collector.CollectModeTiUP
			if reportEnabled {
				teleReport.CommandInfo = &telemetry.CollectInfo{
					ID:         clsID,
					Mode:       collector.CollectModeTiUP,
					ArgYes:     skipConfirm,
					ArgLimit:   cOpt.Limit,
					ArgInclude: inc,
					ArgExclude: ext,
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

	cmd.Flags().StringSliceVarP(&gOpt.Roles, "role", "R", nil, "Only collect data from specified roles")
	cmd.Flags().StringSliceVarP(&gOpt.Nodes, "node", "N", nil, "Only collect data from specified nodes")
	cmd.Flags().Uint64Var(&gOpt.APITimeout, "api-timeout", 60, "Timeout in seconds when querying APIs.")
	cmd.Flags().StringVarP(&opt.ScrapeBegin, "from", "f", time.Now().Add(time.Hour*-2).Format(time.RFC3339), "start timepoint when collecting timeseries data")
	cmd.Flags().StringVarP(&opt.ScrapeEnd, "to", "t", time.Now().Format(time.RFC3339), "stop timepoint when collecting timeseries data")
	cmd.Flags().BoolVar(&collectAll, "all", false, "Collect all data")
	cmd.Flags().StringSliceVar(&inc, "include", []string{"system", "config", "monitor", "log.std"}, "types of data not to collect")
	cmd.Flags().StringSliceVar(&cOpt.MetricsFilter, "metricsfilter", nil, "prefix of metrics to collect")
	cmd.Flags().IntVar(&cOpt.MetricsLimit, "metricslimit", 10000, "metric size limit of single request, specified in series*hour per request")
	cmd.Flags().StringVar(&metricsConf, "metricsconfig", "", "config file of metricsfilter")
	cmd.Flags().StringVarP(&cOpt.Dir, "output", "o", "", "output directory of collected data")
	cmd.Flags().IntVarP(&cOpt.Limit, "limit", "l", -1, "Limits the used bandwidth, specified in Kbit/s")
	cmd.Flags().BoolVar(&cOpt.CompressScp, "compress-scp", true, "Compress when transfer config and logs.Only works with system ssh")
	cmd.Flags().BoolVar(&cOpt.CompressMetrics, "compress-metrics", true, "Compress collected metrics data.")
	cmd.Flags().BoolVar(&cOpt.ExitOnError, "exit-on-error", false, "Stop collecting and exit if an error occurs.")
	cmd.Flags().BoolVar(&cOpt.RawMonitor, "raw-monitor", false, "Collect raw prometheus data")

	return cmd
}
