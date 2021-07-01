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
	"github.com/pingcap/tidb-foresight/collector"
	"github.com/spf13/cobra"
)

func newRebuildCmd() *cobra.Command {
	opt := collector.RebuildOptions{}
	cmd := &cobra.Command{
		Use:   "rebuild <path-to-the-dump>",
		Short: "Rebuild monitoring systems from the dumped data.",
		Long: `Rebuild monitoring systems from the dumped metrics from
a TiDB cluster. Metrics are reloaded to an InfluxDB instance
and can be read from Prometheus with the "remote_read" feature.
The path must be the root directory of a data set dumped by
the diagnostic collector, where the meta.yaml is available.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}

			opt.Concurrency = gOpt.Concurrency
			return collector.LoadMetrics(args[0], &opt)
		},
	}

	cmd.Flags().StringVar(&opt.Host, "host", "localhost", "The host of influxdb.")
	cmd.Flags().IntVar(&opt.Port, "port", 8086, "The port of influxdb.")
	cmd.Flags().StringVar(&opt.User, "user", "", "The username of influxdb.")
	cmd.Flags().StringVar(&opt.Passwd, "passwd", "", "The password of user.")
	cmd.Flags().StringVar(&opt.DBName, "db", "diagcollector", "The database name of imported metrics.")
	cmd.Flags().IntVar(&opt.Chunk, "chunk", 2000, "The chunk size of writing, larger values could speed the process but may timeout.")

	return cmd
}
