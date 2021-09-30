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
	"github.com/pingcap/diag/pkg/packager"
	"github.com/spf13/cobra"
)

var pOpt packager.PackageOptions

func newPackageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package <collected-datadir>",
		Short: "Package collected files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return cmd.Help()
			}
			return packager.PackageCollectedData(&pOpt)
		},
	}

	cmd.Flags().StringVarP(&pOpt.InputDir, "input", "i", "", "input directory of collected data")
	cmd.Flags().StringVarP(&pOpt.OutputFile, "output", "o", "", "output file of packaged data")
	cmd.Flags().StringVar(&pOpt.Compress, "compress", "", "compression algorithm, use: 'gzip', 'zstd'")

	return cmd
}
