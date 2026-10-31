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
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

func TestLiteralIdentityAndIPv6(t *testing.T) {
	r, err := New([]string{"10.1.1.2", "2001:db8:1234::1", "10.1.1.1"})
	require.NoError(t, err)
	input := `10.1.1.1:4000 10.1.1.2 10.1.1.10 [2001:0DB8:1234:0:0:0:0:1]:4000 [2001:db8:1234::1%eth0] ::ffff:10.1.1.1`
	out, err := r.text([]byte(input))
	require.NoError(t, err)
	require.Equal(t, `198.18.0.1:4000 198.18.0.2 10.1.1.10 [2001:db8::1]:4000 [2001:db8::1%eth0] ::ffff:198.18.0.1`, string(out))
	r2, err := New([]string{"10.1.1.1", "10.1.1.2", "2001:db8:1234::1", "10.1.1.1"})
	require.NoError(t, err)
	require.Equal(t, r.mapping, r2.mapping)
}

func TestValidateIPs(t *testing.T) {
	for _, list := range [][]string{nil, {""}, {"1.2.3.999"}, {"host.example"}, {"10.1.1.1:4000"}, {"10.1.1.0/24"}, {"fe80::1%eth0"}} {
		_, err := New(list)
		require.Error(t, err)
	}
}

func TestIPv6FilenameAndAmbiguousLiteral(t *testing.T) {
	r, err := New([]string{"2001:db8:1234::1", "10.1.1.1"})
	require.NoError(t, err)
	out, err := r.text([]byte("2001:db8:1234::1.log peer:2001:db8:1234::1 110.1.1.1 10.1.1.10"))
	require.NoError(t, err)
	require.Equal(t, "2001:db8::1.log peer:2001:db8::1 110.1.1.1 10.1.1.10", string(out))
	// A context that looks like a different IPv6 address must never silently
	// retain the selected address as a literal substring.
	_, err = r.text([]byte("add:2001:db8:1234::1"))
	require.Error(t, err)
}

func TestCopyPathsAndCompressedMetrics(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	input := filepath.Join(root, "data-10.1.1.1")
	require.NoError(t, os.MkdirAll(filepath.Join(input, "logs", "10.1.1.1"), 0700))
	body := []byte(`{"status":"success","data":{"result":[{"metric":{"instance":"10.1.1.1:4000"},"values":[[123,"1"]]}]}}`)
	write := func(name string, b []byte) { require.NoError(t, os.WriteFile(filepath.Join(input, name), b, 0600)) }
	write("cluster.json", []byte(`{"cluster_id":"123","topology":{"host":"10.1.1.1"}}`))
	write("logs/10.1.1.1/tidb.log", []byte("peer=10.1.1.1:4000\n"))
	var metric bytes.Buffer
	zw, err := zstd.NewWriter(&metric)
	require.NoError(t, err)
	_, err = zw.Write(body)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	// Real Diag metrics/alerts can be zstd despite a .json suffix.
	write("metrics.json", metric.Bytes())
	var compressed bytes.Buffer
	gw := gzip.NewWriter(&compressed)
	gw.Name = "10.1.1.1.log"
	gw.Comment = "10.1.1.1"
	_, err = gw.Write([]byte("peer=10.1.1.1\n"))
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	write("tidb.log.gz", compressed.Bytes())
	r, err := New([]string{"10.1.1.1"})
	require.NoError(t, err)
	out, report, err := r.Copy(input, "")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "data-198.18.0.1-redacted"), out)
	require.Equal(t, 4, report.Files)
	require.GreaterOrEqual(t, report.Replacements, 5)
	log, err := os.ReadFile(filepath.Join(out, "logs/198.18.0.1/tidb.log"))
	require.NoError(t, err)
	require.Equal(t, "peer=198.18.0.1:4000\n", string(log))
	original, err := os.ReadFile(filepath.Join(input, "logs/10.1.1.1/tidb.log"))
	require.NoError(t, err)
	require.Contains(t, string(original), "10.1.1.1")
	data, err := os.ReadFile(filepath.Join(out, "metrics.json"))
	require.NoError(t, err)
	zr, err := zstd.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer zr.Close()
	decoded, err := io.ReadAll(zr)
	require.NoError(t, err)
	require.JSONEq(t, strings.ReplaceAll(string(body), "10.1.1.1", "198.18.0.1"), string(decoded))
	data, err = os.ReadFile(filepath.Join(out, "tidb.log.gz"))
	require.NoError(t, err)
	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()
	require.Empty(t, gr.Name)
	require.Empty(t, gr.Comment)
	decoded, err = io.ReadAll(gr)
	require.NoError(t, err)
	require.Equal(t, "peer=198.18.0.1\n", string(decoded))
}

func TestNestedArchives(t *testing.T) {
	var nested bytes.Buffer
	zw := zip.NewWriter(&nested)
	f, err := zw.Create("10.1.1.1.log")
	require.NoError(t, err)
	_, err = f.Write([]byte("addr=10.1.1.1\n"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "10.1.1.1/data.zip", Mode: 0600, Size: int64(nested.Len()), Uname: "10.1.1.1"}))
	_, err = tw.Write(nested.Bytes())
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	// Discover through nested tar/zip readers before generating a copy.
	root := t.TempDir()
	input := filepath.Join(root, "input")
	require.NoError(t, os.Mkdir(input, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(input, "archive.tar"), archive.Bytes(), 0600))
	auto, err := Discover(input)
	require.NoError(t, err)
	_, report, err := auto.Copy(input, filepath.Join(root, "output"))
	require.NoError(t, err)
	require.Equal(t, 1, report.IPs)
	r, err := New([]string{"10.1.1.1"})
	require.NoError(t, err)
	r.remaining = maxExpandedBytes
	var out bytes.Buffer
	require.NoError(t, r.transform(&archive, &out, "archive.tar", 0))
	tr := tar.NewReader(&out)
	header, err := tr.Next()
	require.NoError(t, err)
	require.Equal(t, "198.18.0.1/data.zip", header.Name)
	require.Empty(t, header.Uname)
	data, err := io.ReadAll(tr)
	require.NoError(t, err)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	require.Equal(t, "198.18.0.1.log", zr.File[0].Name)
	zf, err := zr.File[0].Open()
	require.NoError(t, err)
	defer zf.Close()
	data, err = io.ReadAll(zf)
	require.NoError(t, err)
	require.Equal(t, "addr=198.18.0.1\n", string(data))
}

func TestCopyFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"profile.pb", []byte{0, 10, 1, 23}},
		{"data.xz", []byte("compressed")},
		{"bad.gz", []byte{0x1f, 0x8b, 0}},
		{"collision.log", []byte("10.1.1.1 198.18.0.1")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			input, output := filepath.Join(root, "input"), filepath.Join(root, "output")
			require.NoError(t, os.Mkdir(input, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(input, tc.name), tc.data, 0600))
			r, err := New([]string{"10.1.1.1"})
			require.NoError(t, err)
			_, _, err = r.Copy(input, output)
			require.Error(t, err)
			_, err = os.Stat(output)
			require.True(t, os.IsNotExist(err))
			entries, err := os.ReadDir(root)
			require.NoError(t, err)
			require.Len(t, entries, 1)
		})
	}
}

func TestOutputAndSymlinkSafety(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	require.NoError(t, os.Mkdir(input, 0700))
	r, err := New([]string{"10.1.1.1"})
	require.NoError(t, err)
	for _, output := range []string{input, filepath.Join(input, "nested"), root} {
		_, _, err = r.Copy(input, output)
		require.Error(t, err)
	}
	require.NoError(t, os.Symlink(input, filepath.Join(root, "alias")))
	_, _, err = r.Copy(filepath.Join(root, "alias"), filepath.Join(root, "out"))
	require.Error(t, err)
	_, _, err = r.Copy(input, filepath.Join(root, "alias", "nested"))
	require.Error(t, err)
	require.NoError(t, os.Symlink("/etc/passwd", filepath.Join(input, "link")))
	_, _, err = r.Copy(input, filepath.Join(root, "out"))
	require.Error(t, err)
}

func TestChunkBoundaryAndLargeJSON(t *testing.T) {
	r, err := New([]string{"10.1.1.1", "2001:db8:1234::1"})
	require.NoError(t, err)
	// A large JSON document without newlines and a multibyte rune near the cut.
	input := strings.Repeat(`{"x":"中"},`, 10000) + strings.Repeat("x ", (64<<10)-5) + "10.1.1.1 [2001:db8:1234::1]"
	r.remaining = maxExpandedBytes
	var out bytes.Buffer
	require.NoError(t, r.plain(strings.NewReader(input), &out))
	require.Equal(t, strings.NewReplacer("10.1.1.1", "198.18.0.1", "2001:db8:1234::1", "2001:db8::1").Replace(input), out.String())
}

func TestArchiveAndResourceSafety(t *testing.T) {
	for _, name := range []string{"../escape", "/absolute", `dir\escape`} {
		require.Error(t, archiveName(name))
	}
	r, err := New([]string{"10.1.1.1"})
	require.NoError(t, err)
	r.remaining = 2
	require.Error(t, r.plain(strings.NewReader("large"), io.Discard))
	require.Error(t, r.transform(strings.NewReader("x"), io.Discard, "x", maxDepth+1))
}
