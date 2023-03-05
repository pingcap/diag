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
	"github.com/spf13/cobra"
)

func newkubeDumpCmd() *cobra.Command {
	opt := collector.BaseOptions{}
	cOpt := collector.CollectOptions{}
	cOpt.Collectors, _ = collector.ParseCollectTree([]string{collector.CollectTypeMonitor}, nil)
	var (
		clsName  string
		clsID    string
		caPath   string
		certPath string
		keyPath  string
	)

	cmd := &cobra.Command{
		Use:   "kubedump",
		Short: "Dump TSDB files from a Prometheus pod.",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.SetDisplayModeFromString(gOpt.DisplayMode)

			cm := collector.NewManager("tidb", nil, log)

			cOpt.Collectors, _ = collector.ParseCollectTree([]string{"monitor.metric"}, nil)
			opt.Cluster = clsName
			cOpt.RawMonitor = true
			cOpt.RawRequest = strings.Join(os.Args[1:], " ")
			cOpt.Mode = collector.CollectModeManual      // set collect mode
			cOpt.ExtendedAttrs = make(map[string]string) // init attributes map
			cOpt.ExtendedAttrs[collector.AttrKeyClusterID] = clsID
			cOpt.ExtendedAttrs[collector.AttrKeyTLSCAFile] = caPath
			cOpt.ExtendedAttrs[collector.AttrKeyTLSCertFile] = certPath
			cOpt.ExtendedAttrs[collector.AttrKeyTLSKeyFile] = keyPath

			_, err := cm.CollectClusterInfo(&opt, &cOpt, &gOpt, nil, nil, skipConfirm)

			return err
		},
	}

	cmd.Flags().StringVar(&opt.Kubeconfig, "kubeconfig", "", "path of kubeconfig")
	cmd.Flags().StringVar(&clsName, "name", "", "name of the TiDB cluster")
	cmd.Flags().StringVar(&clsID, "cluster-id", "", "ID of the TiDB cluster")
	cmd.Flags().StringVar(&opt.Namespace, "namespace", "", "namespace of prometheus")
	cmd.Flags().StringVar(&cOpt.PodName, "pod", "", "pod name of prometheus")
	cmd.Flags().StringVar(&cOpt.ContainerName, "container", "", "container name of prometheus")
	// cmd.Flags().StringVar(&caPath, "ca-file", "", "path to the CA of TLS enabled cluster")
	// cmd.Flags().StringVar(&certPath, "cert-file", "", "path to the client certification of TLS enabled cluster")
	// cmd.Flags().StringVar(&keyPath, "key-file", "", "path to the private key of client certification of TLS enabled cluster")
	cmd.Flags().StringVarP(&opt.ScrapeBegin, "from", "f", time.Now().Add(time.Hour*-2).Format(time.RFC3339), "start timepoint when collecting timeseries data")
	cmd.Flags().StringVarP(&opt.ScrapeEnd, "to", "t", time.Now().Format(time.RFC3339), "stop timepoint when collecting timeseries data")
	cmd.Flags().StringVarP(&cOpt.Dir, "output", "o", "", "output directory of collected data")

	cobra.MarkFlagRequired(cmd.Flags(), "kubeconfig")
	cobra.MarkFlagRequired(cmd.Flags(), "name")
	cobra.MarkFlagRequired(cmd.Flags(), "cluster-id")
	cobra.MarkFlagRequired(cmd.Flags(), "namespace")
	cobra.MarkFlagRequired(cmd.Flags(), "pod")
	cobra.MarkFlagRequired(cmd.Flags(), "container")

	return cmd
}
