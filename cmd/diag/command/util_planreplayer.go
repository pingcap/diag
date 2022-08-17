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
	"os"
	"strings"
	"time"

	"github.com/pingcap/diag/collector"
	"github.com/pingcap/diag/pkg/telemetry"
	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/spf13/cobra"
)

func newPlanReplayerCmd() *cobra.Command {
	opt := collector.BaseOptions{}
	cOpt := collector.CollectOptions{}
	cOpt.Collectors, _ = collector.ParseCollectTree([]string{collector.CollectTypePlanReplayer}, nil)
	var (
		clsName        string
		tidbHost       string
		tidbPort       string
		tidbStatusPort string
		pdEndpoint     string
		caPath         string
		certPath       string
		keyPath        string
	)

	cmd := &cobra.Command{
		Use:   "planreplayer",
		Short: "Dump SQL plan data for replayer from a TiDB endpoint.",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.SetDisplayModeFromString(gOpt.DisplayMode)
			spec.Initialize("cluster")
			tidbSpec := spec.GetSpecManager()
			cm := collector.NewManager("tidb", tidbSpec, log)

			opt.Cluster = clsName
			opt.ScrapeBegin = time.Now().Add(time.Minute * -1).Format(time.RFC3339)
			opt.ScrapeEnd = time.Now().Format(time.RFC3339)
			cOpt.RawRequest = strings.Join(os.Args[1:], " ")
			cOpt.Mode = collector.CollectModeManual      // set collect mode
			cOpt.ExtendedAttrs = make(map[string]string) // init attributes map
			cOpt.ExtendedAttrs[collector.AttrKeyPDEndpoint] = pdEndpoint
			cOpt.ExtendedAttrs[collector.AttrKeyTiDBHost] = tidbHost
			cOpt.ExtendedAttrs[collector.AttrKeyTiDBPort] = tidbPort
			cOpt.ExtendedAttrs[collector.AttrKeyTiDBStatus] = tidbStatusPort
			cOpt.ExtendedAttrs[collector.AttrKeyTLSCAFile] = caPath
			cOpt.ExtendedAttrs[collector.AttrKeyTLSCertFile] = certPath
			cOpt.ExtendedAttrs[collector.AttrKeyTLSKeyFile] = keyPath

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
			if reportEnabled {
				if size, err := utils.DirSize(cOpt.Dir); err == nil {
					teleReport.CommandInfo.(*telemetry.CollectInfo).
						DataSize = size
				}
			}
			return err
		},
	}

	cmd.Flags().StringVar(&clsName, "name", "", "name of the TiDB cluster")
	cmd.Flags().StringVar(&tidbHost, "tidb-host", "", "host of the TiDB server")
	cmd.Flags().StringVarP(&tidbPort, "tidb-port", "", "4000", "port of the TiDB server")
	cmd.Flags().StringVarP(&tidbStatusPort, "tidb-status", "", "10080", "status port of the TiDB server")
	cmd.Flags().StringVar(&pdEndpoint, "pd", "", "PD endpoint of the TiDB cluster")
	cmd.Flags().StringVar(&caPath, "ca-file", "", "path to the CA of TLS enabled cluster")
	cmd.Flags().StringVar(&certPath, "cert-file", "", "path to the client certification of TLS enabled cluster")
	cmd.Flags().StringVar(&keyPath, "key-file", "", "path to the private key of client certification of TLS enabled cluster")
	cmd.Flags().StringVar(&cOpt.ExplainSQLPath, "explain-sql", "", "File path for explain sql")
	cmd.Flags().StringVarP(&cOpt.Dir, "output", "o", "", "output directory of collected data")
	cobra.MarkFlagRequired(cmd.Flags(), "name")
	cobra.MarkFlagRequired(cmd.Flags(), "tidb-host")
	cobra.MarkFlagRequired(cmd.Flags(), "pd")
	cobra.MarkFlagRequired(cmd.Flags(), "explain-sql")

	return cmd
}
