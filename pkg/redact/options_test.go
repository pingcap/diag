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
	"compress/gzip"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeFixture(t *testing.T, root, name, body string) {
	t.Helper()
	file := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0700))
	require.NoError(t, os.WriteFile(file, []byte(body), 0600))
}

func TestRewriteMatchesCopyAndPreservesReferences(t *testing.T) {
	root := t.TempDir()
	input, work, copyDir := filepath.Join(root, "input"), filepath.Join(root, "work"), filepath.Join(root, "copy")
	writeFixture(t, input, "10.12.34.56/tidb.log", "peer=10.12.34.56:4000 other=10.12.35.57\n")
	writeFixture(t, input, "10.12.35.57/tidb.log", "peer=10.12.35.57:4000\n")
	writeFixture(t, input, "cluster.json", `{"hosts":["10.12.34.56","10.12.35.57"],"log_files":["10.12.34.56/tidb.log","10.12.35.57/tidb.log"]}`)
	f, err := os.Create(filepath.Join(input, "tidb.log.gz"))
	require.NoError(t, err)
	gw := gzip.NewWriter(f)
	_, err = gw.Write([]byte("peer=10.12.34.56\n"))
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())
	require.NoError(t, os.CopyFS(work, os.DirFS(input)))
	copyRedactor, err := Discover(input)
	require.NoError(t, err)
	require.NoError(t, copyRedactor.KeepIPv4Octets(2))
	_, _, err = copyRedactor.Copy(input, copyDir)
	require.NoError(t, err)
	r, err := Discover(work)
	require.NoError(t, err)
	require.NoError(t, r.KeepIPv4Octets(2))
	output, report, err := r.Rewrite(work)
	require.NoError(t, err)
	require.Equal(t, work, output)
	require.Equal(t, 4, report.Files)
	require.NoError(t, CheckInPlaceComplete(work))
	count := 0
	require.NoError(t, filepath.WalkDir(copyDir, func(file string, entry os.DirEntry, err error) error {
		require.NoError(t, err)
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(copyDir, file)
		require.NoError(t, err)
		expected, err := os.ReadFile(file)
		require.NoError(t, err)
		actual, err := os.ReadFile(filepath.Join(work, rel))
		require.NoError(t, err)
		require.Equal(t, expected, actual)
		count++
		return nil
	}))
	require.Equal(t, 4, count)
	_, err = os.Stat(filepath.Join(work, "10.12.34.56"))
	require.True(t, os.IsNotExist(err))
	data, err := os.ReadFile(filepath.Join(work, "cluster.json"))
	require.NoError(t, err)
	var metadata struct {
		Logs []string `json:"log_files"`
	}
	require.NoError(t, json.Unmarshal(data, &metadata))
	for _, name := range metadata.Logs {
		_, err := os.Stat(filepath.Join(work, name))
		require.NoError(t, err)
	}
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 3) // No second full rewrite directory was created.
}

func TestRewritePreflightLeavesOriginalUntouched(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.log", "peer=10.1.1.1\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "z.pb"), []byte{0, 1, 2}, 0600))
	r, err := New([]string{"10.1.1.1"})
	require.NoError(t, err)
	_, _, err = r.Rewrite(root)
	require.ErrorContains(t, err, "preflight")
	data, err := os.ReadFile(filepath.Join(root, "a.log"))
	require.NoError(t, err)
	require.Equal(t, "peer=10.1.1.1\n", string(data))
	require.NoError(t, CheckInPlaceComplete(root))
}

func TestInterruptedRewriteIsBlocked(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	writeFixture(t, input, "a.log", "peer=10.1.1.1\n")
	writeFixture(t, input, "b.log", "peer=10.1.1.2\n")
	r, err := Discover(input)
	require.NoError(t, err)
	require.NoError(t, r.MaskLogIPv4())
	plan, err := r.prepareRewrite(input)
	require.NoError(t, err)
	writeFixture(t, input, "b.log", "peer=10.1.1.2 changed after preflight\n")
	require.ErrorContains(t, r.applyRewrite(input, plan), "changed after preflight")
	data, err := os.ReadFile(filepath.Join(input, "a.log"))
	require.NoError(t, err)
	require.Equal(t, "peer=?\n", string(data))
	require.ErrorContains(t, CheckInPlaceComplete(input), "incomplete")
	_, err = Discover(input)
	require.Error(t, err)
	_, _, err = r.Copy(input, filepath.Join(root, "out"))
	require.Error(t, err)
	_, _, err = r.Rewrite(input)
	require.Error(t, err)
	files, err := os.ReadDir(input)
	require.NoError(t, err)
	for _, file := range files {
		require.False(t, strings.HasPrefix(file.Name(), ".diag-redact-file-"))
	}
}

func TestRewriteNestedNamesAndCollision(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "10.1.1.1/10.1.1.2/10.1.1.3.log", "peer=10.1.1.1\n")
	r, err := Discover(root)
	require.NoError(t, err)
	_, _, err = r.Rewrite(root)
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(root, "198.18.0.1/198.18.0.2/198.18.0.3.log"))
	require.NoError(t, err)
	require.Equal(t, "peer=198.18.0.1\n", string(data))
	other := t.TempDir()
	writeFixture(t, other, "2001:db8::1/tidb.log", "peer=[2001:db8::1]")
	writeFixture(t, other, "2001:db8:0:0:0:0:0:1/tidb.log", "peer=[2001:db8::1]")
	r, err = Discover(other)
	require.NoError(t, err)
	_, _, err = r.Rewrite(other)
	require.ErrorContains(t, err, "collision")
	require.NoError(t, CheckInPlaceComplete(other))
}

func TestKeepIPv4Octets(t *testing.T) {
	for _, tc := range []struct {
		count    int
		expected string
	}{{1, "10.0.0.56"}, {2, "198.18.34.56"}} {
		r, err := New([]string{"10.12.34.56", "fd00::1"})
		require.NoError(t, err)
		require.NoError(t, r.KeepIPv4Octets(tc.count))
		data, err := r.text([]byte("peer=10.12.34.56:4000 ipv6=[fd00::1]"))
		require.NoError(t, err)
		require.Contains(t, string(data), "peer="+tc.expected+":4000")
	}
	r, err := New([]string{"10.12.34.56", "172.16.34.56"})
	require.NoError(t, err)
	require.NoError(t, r.KeepIPv4Octets(2))
	require.Equal(t, r.mapping[netip.MustParseAddr("10.12.34.56")], r.mapping[netip.MustParseAddr("172.16.34.56")])
	require.Error(t, r.KeepIPv4Octets(3))
	r, err = New([]string{"198.18.34.56"})
	require.NoError(t, err)
	require.NoError(t, r.KeepIPv4Octets(2))
	require.Equal(t, netip.MustParseAddr("198.19.34.56"), r.mapping[netip.MustParseAddr("198.18.34.56")])
}

func TestOneOctetFixedPrefixSafety(t *testing.T) {
	for _, addresses := range [][]string{
		{"10.0.0.56"},
		{"172.16.34.56", "10.0.0.56"},
	} {
		r, err := New(addresses)
		require.NoError(t, err)
		before := make(map[netip.Addr]netip.Addr, len(r.mapping))
		for original, replacement := range r.mapping {
			before[original] = replacement
		}
		require.Error(t, r.KeepIPv4Octets(1))
		require.Equal(t, before, r.mapping)
	}
	root := t.TempDir()
	writeFixture(t, root, "172.16.34.56/tidb.log", "peer=172.16.34.56:4000 other=192.168.8.57:20160\n")
	writeFixture(t, root, "cluster.json", `{"hosts":["172.16.34.56","192.168.8.57"],"log_files":["172.16.34.56/tidb.log"]}`)
	r, err := Discover(root)
	require.NoError(t, err)
	require.NoError(t, r.KeepIPv4Octets(1))
	_, _, err = r.Rewrite(root)
	require.NoError(t, err)
	body, err := os.ReadFile(filepath.Join(root, "10.0.0.56/tidb.log"))
	require.NoError(t, err)
	require.Equal(t, "peer=10.0.0.56:4000 other=10.0.0.57:20160\n", string(body))
	body, err = os.ReadFile(filepath.Join(root, "cluster.json"))
	require.NoError(t, err)
	require.JSONEq(t, `{"hosts":["10.0.0.56","10.0.0.57"],"log_files":["10.0.0.56/tidb.log"]}`, string(body))
}

func TestRepeatedIPv4SuffixesCopyRewriteAndMapping(t *testing.T) {
	for _, count := range []int{1, 2} {
		for _, inPlace := range []bool{false, true} {
			root := t.TempDir()
			input := filepath.Join(root, "input")
			writeFixture(t, input, "node-a/tidb.log", "peer=10.12.34.56:4000\n")
			writeFixture(t, input, "node-b/tidb.log", "peer=172.16.34.56:4000\n")
			r, err := Discover(input)
			require.NoError(t, err)
			require.NoError(t, r.KeepIPv4Octets(count))
			mapFile := filepath.Join(root, "private", "map.json")
			require.NoError(t, r.SaveMapping(mapFile, input, input+"-redacted"))
			var output string
			if inPlace {
				output, _, err = r.Rewrite(input)
			} else {
				output, _, err = r.Copy(input, input+"-redacted")
			}
			require.NoError(t, err)
			wantIP := "10.0.0.56"
			if count == 2 {
				wantIP = "198.18.34.56"
			}
			for _, node := range []string{"node-a", "node-b"} {
				body, err := os.ReadFile(filepath.Join(output, node, "tidb.log"))
				require.NoError(t, err)
				require.Equal(t, "peer="+wantIP+":4000\n", string(body))
			}
			body, err := os.ReadFile(mapFile)
			require.NoError(t, err)
			var payload struct {
				Entries []mappingEntry `json:"entries"`
			}
			require.NoError(t, json.Unmarshal(body, &payload))
			require.Equal(t, []mappingEntry{
				{Original: "10.12.34.56", Redacted: wantIP},
				{Original: "172.16.34.56", Redacted: wantIP},
			}, payload.Entries)
		}
	}
}

func TestRepeatedIPv4SuffixesStillRejectPathCollisions(t *testing.T) {
	for _, inPlace := range []bool{false, true} {
		input := filepath.Join(t.TempDir(), "input")
		for _, ip := range []string{"10.12.34.56", "172.16.34.56"} {
			writeFixture(t, input, ip+"/tidb.log", "peer="+ip+"\n")
		}
		r, err := Discover(input)
		require.NoError(t, err)
		require.NoError(t, r.KeepIPv4Octets(2))
		if inPlace {
			_, _, err = r.Rewrite(input)
		} else {
			_, _, err = r.Copy(input, input+"-redacted")
		}
		require.ErrorContains(t, err, "path collision")
		require.NoError(t, CheckInPlaceComplete(input))
		for _, ip := range []string{"10.12.34.56", "172.16.34.56"} {
			body, err := os.ReadFile(filepath.Join(input, ip, "tidb.log"))
			require.NoError(t, err)
			require.Equal(t, "peer="+ip+"\n", string(body))
		}
		_, err = os.Lstat(input + "-redacted")
		require.True(t, os.IsNotExist(err))
	}
}

func TestPrivateMappingLocationPermissionsAndContents(t *testing.T) {
	root := t.TempDir()
	input, output := filepath.Join(root, "input"), filepath.Join(root, "output")
	writeFixture(t, input, "tidb.log", "peer=10.12.34.56")
	r, err := Discover(input)
	require.NoError(t, err)
	require.NoError(t, r.KeepIPv4Octets(2))
	file := filepath.Join(root, "ip-map.json")
	require.NoError(t, r.SaveMapping(file, input, output))
	info, err := os.Stat(file)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	var payload struct {
		Entries []mappingEntry `json:"entries"`
		Kept    int            `json:"kept_ipv4_octets"`
	}
	require.NoError(t, json.Unmarshal(data, &payload))
	require.Equal(t, 2, payload.Kept)
	require.Equal(t, []mappingEntry{{Original: "10.12.34.56", Redacted: "198.18.34.56"}}, payload.Entries)
	require.Error(t, r.SaveMapping(file, input, output))
	require.Error(t, r.SaveMapping(filepath.Join(input, "map.json"), input, output))
	require.NoError(t, os.Mkdir(output, 0700))
	require.Error(t, r.SaveMapping(filepath.Join(output, "map.json"), input, output))
	require.NoError(t, os.Symlink(input, filepath.Join(root, "alias")))
	require.Error(t, r.SaveMapping(filepath.Join(root, "alias", "map.json"), input, output))
}

func TestPrivateMappingCreatesMissingParents(t *testing.T) {
	root := t.TempDir()
	input, output := filepath.Join(root, "input"), filepath.Join(root, "output")
	writeFixture(t, input, "tidb.log", "peer=10.12.34.56")
	r, err := Discover(input)
	require.NoError(t, err)
	file := filepath.Join(root, "private", "nested", "ip-map.json")
	require.NoError(t, r.SaveMapping(file, input, output))
	for _, dir := range []string{filepath.Dir(file), filepath.Dir(filepath.Dir(file))} {
		info, err := os.Stat(dir)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0700), info.Mode().Perm())
	}
	info, err := os.Stat(file)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	body, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Contains(t, string(body), `"original_ip": "10.12.34.56"`)
	require.Error(t, r.SaveMapping(file, input, output))
	after, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, body, after)
}

func TestPrivateMappingRejectsUnsafeMissingParents(t *testing.T) {
	root := t.TempDir()
	input, output := filepath.Join(root, "input"), filepath.Join(root, "output")
	writeFixture(t, input, "tidb.log", "peer=10.12.34.56")
	r, err := Discover(input)
	require.NoError(t, err)
	alias := filepath.Join(root, "alias")
	require.NoError(t, os.Symlink(input, alias))
	for _, parent := range []string{input, output, alias} {
		file := filepath.Join(parent, "private", "nested", "map.json")
		require.ErrorContains(t, r.SaveMapping(file, input, output), "must be outside")
		_, err := os.Lstat(filepath.Join(parent, "private"))
		require.True(t, os.IsNotExist(err), file)
	}
	// Dangling links must not cause directories to be created at their targets.
	danglingTarget := filepath.Join(root, "absent")
	dangling := filepath.Join(root, "dangling")
	require.NoError(t, os.Symlink(danglingTarget, dangling))
	require.Error(t, r.SaveMapping(filepath.Join(dangling, "nested", "map.json"), input, output))
	require.Error(t, r.SaveMapping(dangling, input, output))
	_, err = os.Lstat(danglingTarget)
	require.True(t, os.IsNotExist(err))
	// A file used as a directory produces an error and retains its contents.
	blocked := filepath.Join(root, "blocked")
	require.NoError(t, os.WriteFile(blocked, []byte("keep"), 0600))
	require.Error(t, r.SaveMapping(filepath.Join(blocked, "map.json"), input, output))
	body, err := os.ReadFile(blocked)
	require.NoError(t, err)
	require.Equal(t, "keep", string(body))
}

func TestPrivateMappingExistingParentAndConcurrentWriters(t *testing.T) {
	root := t.TempDir()
	input, output := filepath.Join(root, "input"), filepath.Join(root, "output")
	writeFixture(t, input, "tidb.log", "peer=10.12.34.56")
	r, err := Discover(input)
	require.NoError(t, err)
	parent := filepath.Join(root, "existing")
	require.NoError(t, os.Mkdir(parent, 0755))
	require.NoError(t, os.Chmod(parent, 0755))
	file := filepath.Join(parent, "nested", "map.json")
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { results <- r.SaveMapping(file, input, output) }()
	}
	wins := 0
	for i := 0; i < 2; i++ {
		if <-results == nil {
			wins++
		}
	}
	require.Equal(t, 1, wins)
	info, err := os.Stat(parent)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0755), info.Mode().Perm())
	body, err := os.ReadFile(file)
	require.NoError(t, err)
	require.True(t, json.Valid(body))
	require.Contains(t, string(body), `"original_ip": "10.12.34.56"`)
	temps, err := filepath.Glob(filepath.Join(filepath.Dir(file), ".diag-ip-map-*"))
	require.NoError(t, err)
	require.Empty(t, temps)
}
