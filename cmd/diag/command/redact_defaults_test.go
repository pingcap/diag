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
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRedactDefaultsAreSeparateAndOverridable(t *testing.T) {
	for _, makeCmd := range []func() *cobra.Command{newCollectCmd, newCollectkCmd, newCollectDMCmd, newRedactCmd} {
		original := makeCmd()
		require.Equal(t, "false", original.Flags().Lookup("redact-all-ips").Value.String())
		require.Equal(t, "false", original.Flags().Lookup("redact-in-place").Value.String())
		require.Equal(t, "0", original.Flags().Lookup("keep-ipv4-octets").Value.String())
		root := &cobra.Command{}
		cmd := makeCmd()
		root.AddCommand(cmd)
		require.NoError(t, setRedactionDefaults(root))
		for name, want := range map[string]string{"redact-all-ips": "true", "redact-in-place": "true", "keep-ipv4-octets": "2"} {
			flag := cmd.Flags().Lookup(name)
			require.Equal(t, want, flag.Value.String())
			require.Equal(t, want, flag.DefValue)
			require.False(t, flag.Changed)
		}
		require.Equal(t, "N", cmd.Flags().Lookup("ip-map-output").Value.String())
		if cmd.Name() != "redact" {
			require.Equal(t, "true", cmd.Flags().Lookup("exit-on-error").Value.String())
			require.NoError(t, cmd.Flags().Set("exit-on-error", "false"))
			require.Equal(t, "false", cmd.Flags().Lookup("exit-on-error").Value.String())
		}
		require.NoError(t, cmd.PreRunE(cmd, nil))
		require.NoError(t, cmd.Flags().Set("redact-in-place", "false"))
		require.NoError(t, cmd.Flags().Set("keep-ipv4-octets", "0"))
		require.NoError(t, cmd.Flags().Set("redact-output", "separate-output"))
		require.NoError(t, cmd.PreRunE(cmd, nil))
		// Constructing the separate component never changes the normal factory.
		require.Equal(t, "false", makeCmd().Flags().Lookup("redact-all-ips").Value.String())
	}
}
