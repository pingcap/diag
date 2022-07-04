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
	"os"

	"github.com/pingcap/diag/pkg/packager"
	"github.com/pingcap/diag/pkg/telemetry"
	"github.com/pingcap/diag/pkg/utils"
	"github.com/spf13/cobra"
)

func newPackageCmd() *cobra.Command {
	pOpt := &packager.PackageOptions{}
	cmd := &cobra.Command{
		Use:   "package <collected-datadir>",
		Short: "Package collected files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return cmd.Help()
			}

			// If both specified, the dir path in -i/--input argument has
			// higher priority
			if len(args) == 1 && pOpt.InputDir == "" {
				pOpt.InputDir = args[0]
			}

			if diagConfig.Clinic.Region == "" {
				err := diagConfig.InteractiveSetRegion()
				if err != nil {
					return err
				}
				err = diagConfig.Save()
				if err != nil {
					return err
				}
			}
			pOpt.Cert = diagConfig.Clinic.Region.Cert()

			if reportEnabled {
				inputSize, _ := utils.DirSize(pOpt.InputDir)
				teleReport.CommandInfo = &telemetry.PackageInfo{
					OriginalSize: inputSize,
				}
			}

			f, err := packager.PackageCollectedData(pOpt, skipConfirm)
			if err != nil {
				return err
			}
			log.Infof("packaged data set saved to %s", f)

			if reportEnabled {
				fi, err := os.Stat(f)
				if err == nil {
					teleReport.CommandInfo.(*telemetry.PackageInfo).
						PackageSize = fi.Size()
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&pOpt.InputDir, "input", "i", "", "input directory of collected data")
	cmd.Flags().StringVarP(&pOpt.OutputFile, "output", "o", "", "output file of packaged data")
	cmd.Flags().BoolVar(&pOpt.Rebuild, "rebuild", true, "rebuild package immediately after upload")

	return cmd
}
