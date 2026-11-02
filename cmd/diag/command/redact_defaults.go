// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"fmt"
	"os"

	"github.com/pingcap/tiup/pkg/tui"
	"github.com/spf13/cobra"
)

// ExecuteRedact runs the separate local component. The normal Diag entry
// point never calls this, so its defaults and configuration profile stay intact.
func ExecuteRedact() {
	componentName = "diag-redact"
	tui.RegisterArg0("tiup diag-redact")
	rootCmd.Use = "tiup diag-redact"
	rootCmd.SetVersionTemplate("tiup diag-redact {{.Version}}\n")
	if err := setRedactionDefaults(rootCmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	Execute()
}

func setRedactionDefaults(root *cobra.Command) error {
	for _, cmd := range root.Commands() {
		switch cmd.Name() {
		case "collect", "collectk", "collectdm", "redact":
		default:
			continue
		}
		defaults := []struct{ name, value string }{
			{"redact-all-ips", "true"},
			{"keep-ipv4-octets", "2"},
			{"redact-in-place", "true"},
		}
		if cmd.Name() != "redact" {
			defaults = append(defaults, struct{ name, value string }{"exit-on-error", "true"})
		}
		for _, setting := range defaults {
			flag := cmd.Flags().Lookup(setting.name)
			if flag == nil {
				return fmt.Errorf("%s is missing flag %s", cmd.Name(), setting.name)
			}
			// Set the bound value and help default without marking it as a user
			// selection. Explicit CLI flags still override these defaults.
			if err := flag.Value.Set(setting.value); err != nil {
				return err
			}
			flag.DefValue = setting.value
		}
	}
	return nil
}
