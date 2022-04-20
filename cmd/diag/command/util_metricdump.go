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
	"bufio"
	"os"
	"strings"
	"time"

	"github.com/pingcap/diag/collector"
	"github.com/pingcap/diag/pkg/telemetry"
	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/spf13/cobra"
)

func newMetricDumpCmd() *cobra.Command {
	opt := collector.BaseOptions{}
	cOpt := collector.CollectOptions{
		Include: set.NewStringSet(collector.CollectTypeMonitor),
		Exclude: set.NewStringSet(),
	}
	var (
		clsName      string
		promEndpoint string
		pdEndpoint   string
		metricsConf  string
		caPath       string
		certPath     string
		keyPath      string
	)

	cmd := &cobra.Command{
		Use:   "metricdump",
		Short: "Dump metrics from a Prometheus endpoint.",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.SetDisplayModeFromString(gOpt.DisplayMode)
			spec.Initialize("cluster")
			tidbSpec := spec.GetSpecManager()
			cm := collector.NewManager("tidb", tidbSpec, log)

			opt.Cluster = clsName
			cOpt.RawRequest = strings.Join(os.Args[1:], " ")
			cOpt.Mode = collector.CollectModeManual      // set collect mode
			cOpt.ExtendedAttrs = make(map[string]string) // init attributes map
			cOpt.ExtendedAttrs[collector.AttrKeyPDEndpoint] = pdEndpoint
			cOpt.ExtendedAttrs[collector.AttrKeyPromEndpoint] = promEndpoint
			cOpt.ExtendedAttrs[collector.AttrKeyTLSCAFile] = caPath
			cOpt.ExtendedAttrs[collector.AttrKeyTLSCertFile] = certPath
			cOpt.ExtendedAttrs[collector.AttrKeyTLSKeyFile] = keyPath

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

			if reportEnabled {
				clsID := scrubClusterName(opt.Cluster)
				teleCommand = append(teleCommand, clsID)
				teleReport.CommandInfo = &telemetry.CollectInfo{
					ID:       clsID,
					Mode:     cOpt.Mode,
					ArgYes:   skipConfirm,
					ArgLimit: cOpt.Limit,
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

	cmd.Flags().StringVar(&clsName, "name", "", "name of the TiDB cluster")
	cmd.Flags().StringVar(&promEndpoint, "prometheus", "", "Prometheus endpoint")
	cmd.Flags().StringVar(&pdEndpoint, "pd", "", "PD endpoint of the TiDB cluster")
	cmd.Flags().StringVar(&caPath, "ca-file", "", "path to the CA of TLS enabled cluster")
	cmd.Flags().StringVar(&certPath, "cert-file", "", "path to the client certification of TLS enabled cluster")
	cmd.Flags().StringVar(&keyPath, "key-file", "", "path to the private key of client certification of TLS enabled cluster")
	cmd.Flags().StringVarP(&opt.ScrapeBegin, "from", "f", time.Now().Add(time.Hour*-2).Format(time.RFC3339), "start timepoint when collecting timeseries data")
	cmd.Flags().StringVarP(&opt.ScrapeEnd, "to", "t", time.Now().Format(time.RFC3339), "stop timepoint when collecting timeseries data")
	cmd.Flags().StringSliceVar(&cOpt.MetricsFilter, "metricsfilter", nil, "prefix of metrics to collect")
	cmd.Flags().StringVar(&metricsConf, "metricsconfig", "", "config file of metricsfilter")
	cmd.Flags().StringVarP(&cOpt.Dir, "output", "o", "", "output directory of collected data")

	cobra.MarkFlagRequired(cmd.Flags(), "name")
	cobra.MarkFlagRequired(cmd.Flags(), "prometheus")
	cobra.MarkFlagRequired(cmd.Flags(), "pd")

	return cmd
}
