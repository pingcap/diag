// Copyright 2023 PingCAP, Inc.
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
	"reflect"
	"strings"
	"time"

	"github.com/pingcap/diag/collector"
	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/tiup/pkg/tui"
	tiuputils "github.com/pingcap/tiup/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func newCollectkCmd() *cobra.Command {
	var collectAll bool
	var direct bool
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
		Use:   "collectk <cluster-name>",
		Short: "(EXPERIMENTAL) Collect information and metrics from the tidb-operator deployed cluster.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}
			cOpt.DiagMode = collector.DiagModeCmd
			cOpt.UsePortForward = !direct
			cOpt.RawRequest = strings.Join(os.Args[1:], " ")

			log.SetDisplayModeFromString(gOpt.DisplayMode)
			cm := collector.NewManager("tidb", nil, log)

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

			cOpt.Mode = collector.CollectModeK8s

			cfg, err := clientcmd.BuildConfigFromFlags("", opt.Kubeconfig)
			if err != nil {
				return err
			}

			kubeCli, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return fmt.Errorf("failed to get kubernetes Clientset: %v", err)
			}
			dynCli, err := dynamic.NewForConfig(cfg)
			if err != nil {
				return fmt.Errorf("failed to get kubernetes dynamic client interface: %v", err)
			}

			_, err = cm.CollectClusterInfo(&opt, &cOpt, &gOpt, kubeCli, dynCli, skipConfirm)
			return err
		},
	}

	cmd.Flags().StringVar(&cOpt.ProfileName, "profile", "", "File name of a pre-defined collecting profile")
	cmd.Flags().StringSliceVarP(&gOpt.Roles, "role", "R", nil, "Only collect data from specified roles")
	// cmd.Flags().StringSliceVarP(&gOpt.Nodes, "node", "N", nil, "Only collect data from specified nodes")
	cmd.Flags().StringVarP(&opt.ScrapeBegin, "from", "f", time.Now().Add(time.Hour*-2).Format(time.RFC3339), "start timepoint when collecting timeseries data")
	cmd.Flags().StringVarP(&opt.ScrapeEnd, "to", "t", time.Now().Format(time.RFC3339), "stop timepoint when collecting timeseries data")
	cmd.Flags().BoolVar(&collectAll, "all", false, "Collect all data")
	cmd.Flags().StringSliceVar(&inc, "include", []string{"monitor.metric", "log.std", "log.slow"}, "types of data to collect")
	cmd.Flags().StringSliceVar(&ext, "exclude", nil, "types of data not to collect")
	cmd.Flags().StringSliceVar(&cOpt.MetricsFilter, "metricsfilter", nil, "prefix of metrics to collect")
	cmd.Flags().IntVar(&cOpt.MetricsLimit, "metricslimit", 10000, "metric size limit of single request, specified in series*hour per request")
	cmd.Flags().StringVar(&metricsConf, "metricsconfig", "", "config file of metricsfilter")
	cmd.Flags().StringVarP(&cOpt.Dir, "output", "o", "", "output directory of collected data")
	// cmd.Flags().IntVarP(&cOpt.Limit, "limit", "l", -1, "Limits the used bandwidth, specified in Kbit/s")
	// cmd.Flags().IntVar(&cOpt.PerfDuration, "perf-duration", 30, "Duration of the collection of profile information in seconds")
	cmd.Flags().Uint64Var(&gOpt.APITimeout, "api-timeout", 10, "Timeout in seconds when querying PD APIs.")
	// cmd.Flags().BoolVar(&cOpt.CompressScp, "compress-scp", true, "Compress when transfer config and logs.Only works with system ssh")
	cmd.Flags().BoolVar(&cOpt.CompressMetrics, "compress-metrics", true, "Compress collected metrics data.")
	cmd.Flags().BoolVar(&cOpt.ExitOnError, "exit-on-error", false, "Stop collecting and exit if an error occurs.")
	// cmd.Flags().BoolVar(&cOpt.RawMonitor, "raw-monitor", false, "Collect raw prometheus data")
	// cmd.Flags().StringVar(&cOpt.ExplainSQLPath, "explain-sql", "", "File path for explain sql")
	// cmd.Flags().StringVar(&cOpt.CurrDB, "db", "", "default db for plan replayer collector")

	cmd.Flags().StringVar(&opt.Kubeconfig, "kubeconfig", clientcmd.RecommendedHomeFile, "path of kubeconfig")
	cmd.Flags().StringVar(&opt.Namespace, "namespace", "", "namespace of prometheus")
	cmd.Flags().BoolVar(&direct, "--direct", false, "not use port-forward to collect from inside of k8s cluster")

	return cmd
}
