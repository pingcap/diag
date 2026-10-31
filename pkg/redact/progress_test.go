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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProgressCopyAndRewrite(t *testing.T) {
	for _, inPlace := range []bool{false, true} {
		t.Run(map[bool]string{false: "copy", true: "in-place"}[inPlace], func(t *testing.T) {
			input := filepath.Join(t.TempDir(), "input")
			writeFixture(t, input, "tidb.log", "peer=10.12.34.56:4000\n")
			writeFixture(t, input, "empty.log", "")
			f, err := os.Create(filepath.Join(input, "slow.log.gz"))
			require.NoError(t, err)
			gz := gzip.NewWriter(f)
			_, err = io.WriteString(gz, strings.Repeat("peer=10.12.34.56\n", 100))
			require.NoError(t, err)
			require.NoError(t, gz.Close())
			require.NoError(t, f.Close())
			var events []Progress
			r, err := DiscoverWithProgress(input, func(p Progress) { events = append(events, p) })
			require.NoError(t, err)
			var output string
			if inPlace {
				output, _, err = r.Rewrite(input)
			} else {
				output, _, err = r.Copy(input, input+"-redacted")
			}
			require.NoError(t, err)
			body, err := os.ReadFile(filepath.Join(output, "tidb.log"))
			require.NoError(t, err)
			require.Equal(t, "peer=198.18.0.1:4000\n", string(body))
			var phases []string
			for i, event := range events {
				if i > 0 && event.Phase == events[i-1].Phase {
					require.GreaterOrEqual(t, event.Bytes, events[i-1].Bytes)
					require.GreaterOrEqual(t, event.Files, events[i-1].Files)
				}
				if event.Done {
					phases = append(phases, event.Phase)
					require.Equal(t, event.TotalBytes, event.Bytes, event.Phase)
					require.Equal(t, event.TotalFiles, event.Files, event.Phase)
				}
			}
			want := []string{"Scanning IPs", "Redacting", "Verifying"}
			if inPlace {
				want = []string{"Scanning IPs", "Checking in-place rewrite", "Redacting in place", "Verifying"}
			}
			require.Equal(t, want, phases)
		})
	}
}

func TestProgressEmptyAndFailure(t *testing.T) {
	input := t.TempDir()
	var events []Progress
	callback := func(p Progress) { events = append(events, p) }
	r, err := DiscoverWithProgress(input, callback)
	require.NoError(t, err)
	_, _, err = r.Copy(input, input+"-output")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(input + "-output") })
	require.True(t, events[len(events)-1].Done)
	require.Zero(t, events[len(events)-1].TotalBytes)

	writeFixture(t, input, "invalid.gz", "not gzip")
	events = nil
	r, err = New([]string{"10.12.34.56"})
	require.NoError(t, err)
	r.SetProgress(callback)
	_, _, err = r.Copy(input, input+"-failed")
	require.Error(t, err)
	require.NotEmpty(t, events)
	for _, event := range events {
		require.False(t, event.Done)
	}
	_, err = os.Stat(input + "-failed")
	require.True(t, os.IsNotExist(err))
	body, err := os.ReadFile(filepath.Join(input, "invalid.gz"))
	require.NoError(t, err)
	require.Equal(t, "not gzip", string(body))
	// Disabling the observer restores the original path without a metadata walk.
	r.SetProgress(nil)
	require.Nil(t, r.progress)
}

func TestProgressReportsBytesBeforeFileCompletes(t *testing.T) {
	var events []Progress
	r, err := New([]string{"10.12.34.56"})
	require.NoError(t, err)
	r.SetProgress(func(p Progress) { events = append(events, p) })
	r.progress.Phase = "Redacting"
	reader := r.progressReader(strings.NewReader("abcdefgh"))
	buf := make([]byte, 4)
	n, err := reader.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	require.EqualValues(t, 4, events[0].Bytes)
	require.Zero(t, events[0].Files)
	r.progress.last = time.Now().Add(-time.Second)
	_, err = reader.Read(buf)
	require.NoError(t, err)
	require.EqualValues(t, 8, events[1].Bytes)
	require.False(t, events[1].Done)
}
