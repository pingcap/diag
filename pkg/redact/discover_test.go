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

package redact

import (
	"archive/zip"
	"bytes"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

func TestAutomaticDiscoveryAcrossContentsAndPaths(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input-10.1.1.1")
	require.NoError(t, os.MkdirAll(filepath.Join(input, "10.1.1.2"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(input, "10.1.1.2", "slow.log"), []byte("peer=10.1.1.1:4000 sql='10.1.1.3' invalid=999.1.1.1 phone=13800000000 v6=[::1] any=[::] full=[2001:0DB8:0001:0:0:0:0:2]\n"), 0600))
	var metric bytes.Buffer
	encoder, err := zstd.NewWriter(&metric)
	require.NoError(t, err)
	_, err = encoder.Write([]byte(`{"instance":"198.18.0.1:9090"}`))
	require.NoError(t, err)
	require.NoError(t, encoder.Close())
	require.NoError(t, os.WriteFile(filepath.Join(input, "metrics.json"), metric.Bytes(), 0600))
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	entry, err := zw.Create("10.1.1.4/log.txt")
	require.NoError(t, err)
	_, err = entry.Write([]byte("peer=10.1.1.5\n"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, os.WriteFile(filepath.Join(input, "nested.zip"), archive.Bytes(), 0600))
	r, err := Discover(input)
	require.NoError(t, err)
	require.Len(t, r.mapping, 9)
	for _, ip := range []string{"10.1.1.1", "10.1.1.2", "10.1.1.3", "10.1.1.4", "10.1.1.5", "198.18.0.1", "::", "::1", "2001:db8:1::2"} {
		require.Contains(t, r.mapping, netip.MustParseAddr(ip))
	}
	// A real input address in the placeholder range must itself be replaced.
	for _, replacement := range r.mapping {
		require.NotEqual(t, netip.MustParseAddr("198.18.0.1"), replacement)
	}
	output, report, err := r.Copy(input, filepath.Join(root, "output"))
	require.NoError(t, err)
	require.Equal(t, 9, report.IPs)
	require.Equal(t, 3, report.Files)
	reread, err := Discover(output)
	require.NoError(t, err)
	for ip := range reread.mapping {
		require.True(t, r.destinations[ip])
	}
	logPath := filepath.Join(output, r.mapping[netip.MustParseAddr("10.1.1.2")].String(), "slow.log")
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "999.1.1.1")
	require.Contains(t, string(data), "13800000000")
	require.NotContains(t, string(data), "10.1.1.1")
}

func TestAutomaticModeRefusesChangedInputs(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	require.NoError(t, os.Mkdir(input, 0700))
	log := filepath.Join(input, "log")
	require.NoError(t, os.WriteFile(log, []byte("peer=10.1.1.1"), 0600))
	r, err := Discover(input)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(log, []byte("peer=10.1.1.1 new=10.1.1.2"), 0600))
	output := filepath.Join(root, "output")
	_, _, err = r.Copy(input, output)
	require.ErrorContains(t, err, "unrecognized IP")
	_, err = os.Stat(output)
	require.True(t, os.IsNotExist(err))
}

func TestAutomaticModeEmptyAndUnsupportedData(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	require.NoError(t, os.Mkdir(input, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(input, "log"), []byte("no addresses"), 0600))
	r, err := Discover(input)
	require.NoError(t, err)
	_, report, err := r.Copy(input, filepath.Join(root, "output"))
	require.NoError(t, err)
	require.Zero(t, report.IPs)
	require.Zero(t, report.Replacements)
	require.NoError(t, os.WriteFile(filepath.Join(input, "profile.pb"), []byte{0, 1, 2}, 0600))
	_, err = Discover(input)
	require.Error(t, err)
}
