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
	"fmt"
	"os"

	"github.com/pingcap/tidb-foresight/collector"
	"github.com/pingcap/tiup/pkg/tui/progress"
	"github.com/spf13/cobra"
)

func newRebuildCmd() *cobra.Command {
	opt := collector.RebuildOptions{}
	cmd := &cobra.Command{
		Use:   "rebuild <cluster-name> <path-to-metrics-dump>",
		Short: "Rebuild monitoring systems from the dumped metrics.",
		Long: `Rebuild monitoring systems from the dumped metrics from
a TiDB cluster. Metrics are reloaded to an InfluxDB instance
and can be read from Prometheus with the "remote_read" feature.
A cluster name must be set to identify the data.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return cmd.Help()
			}
			opt.Cluster = args[1]
			dataDir := args[2]
			files, err := os.ReadDir(dataDir)
			if err != nil {
				return err
			}

			// TODO: add progress bar
			b := progress.NewSingleBar("Loading metrics")
			b.StartRenderLoop()
			defer b.StopRenderLoop()

			cnt := 0
			for _, file := range files {
				if file.IsDir() {
					continue
				}
				cnt++
				b.UpdateDisplay(&progress.DisplayProps{
					Prefix: "Loading metrics",
					Suffix: fmt.Sprintf("(%d): %s", cnt, file.Name()),
				})
				f, err := os.Open(file.Name())
				if err != nil {
					b.UpdateDisplay(&progress.DisplayProps{
						Prefix: "Load metrics",
						Suffix: err.Error(),
						Mode:   progress.ModeError,
					})
					return err
				}
				defer f.Close()
				fOpt := opt
				fOpt.File = f
				if err := fOpt.Load(); err != nil {
					b.UpdateDisplay(&progress.DisplayProps{
						Prefix: "Load metrics",
						Suffix: err.Error(),
						Mode:   progress.ModeError,
					})
					return err
				}
			}

			b.UpdateDisplay(&progress.DisplayProps{
				Prefix: "Load metrics",
				Mode:   progress.ModeDone,
			})

			return nil
		},
	}

	cmd.Flags().StringVar(&opt.Host, "host", "localhost", "The host of influxdb.")
	cmd.Flags().IntVar(&opt.Port, "port", 8086, "The port of influxdb.")
	cmd.Flags().StringVar(&opt.User, "user", "", "The username of influxdb.")
	cmd.Flags().StringVar(&opt.Passwd, "passwd", "", "The password of user.")
	cmd.Flags().StringVar(&opt.DBName, "db", "diagcollector", "The database name of imported metrics.")
	cmd.Flags().IntVar(&opt.Chunk, "chunk", 2000, "The chunk size of writing.")

	return cmd
}
