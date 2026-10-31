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

package packager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pingcap/diag/pkg/redact"
	"github.com/stretchr/testify/require"
)

func TestIncompleteRewriteCannotBePackagedIncludingAutoYes(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "diag-fixture")
	require.NoError(t, os.Mkdir(input, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(input, "cluster.json"), []byte(`{"cluster_id":"123"}`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(input, redact.InPlaceMarker), []byte("incomplete"), 0600))
	_, err := selectInputDir(input, true)
	require.ErrorContains(t, err, "incomplete")
	t.Chdir(root)
	_, err = selectInputDir("", true)
	require.ErrorContains(t, err, "incomplete")
	output := filepath.Join(root, "result.diag")
	_, err = PackageCollectedData(&PackageOptions{InputDir: input, OutputFile: output}, true)
	require.ErrorContains(t, err, "incomplete")
	_, err = os.Stat(output)
	require.True(t, os.IsNotExist(err))
}
