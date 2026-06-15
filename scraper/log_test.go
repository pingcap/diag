package scraper

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriteFilteredLogIncludesOnlyTimeRange(t *testing.T) {
	start := mustParseLogTime(t, "2026/05/26 10:00:00.000 +08:00")
	end := mustParseLogTime(t, "2026/05/26 10:02:00.000 +08:00")
	input := strings.Join([]string{
		`[2026/05/26 09:59:59.999 +08:00] [INFO] [test.go:1] ["before"]`,
		`before continuation`,
		`[2026/05/26 10:00:00.000 +08:00] [INFO] [test.go:2] ["at start"]`,
		`start continuation`,
		`[2026/05/26 10:01:00.000 +08:00] [WARN] [test.go:3] ["inside"]`,
		`[2026/05/26 10:02:00.000 +08:00] [ERROR] [test.go:4] ["at end"]`,
		`[2026/05/26 10:02:00.001 +08:00] [INFO] [test.go:5] ["after"]`,
	}, "\n") + "\n"

	var out bytes.Buffer
	written, err := writeFilteredLog(strings.NewReader(input), &out, LogTypeStd, start, end)
	require.NoError(t, err)
	require.True(t, written)

	result := out.String()
	require.NotContains(t, result, "before")
	require.Contains(t, result, "at start")
	require.Contains(t, result, "start continuation")
	require.Contains(t, result, "inside")
	require.Contains(t, result, "at end")
	require.NotContains(t, result, "after")
}

func TestLogScraperOutputsFilteredLogAndOriginalTarget(t *testing.T) {
	start := mustParseLogTime(t, "2026/05/26 10:00:00.000 +08:00")
	end := mustParseLogTime(t, "2026/05/26 10:01:00.000 +08:00")
	dir := t.TempDir()
	logPath := filepath.Join(dir, "tidb.log")
	require.NoError(t, os.WriteFile(logPath, []byte(strings.Join([]string{
		`[2026/05/26 09:59:59.000 +08:00] [INFO] [test.go:1] ["before"]`,
		`[2026/05/26 10:00:00.000 +08:00] [INFO] [test.go:2] ["at start"]`,
		`[2026/05/26 10:01:00.000 +08:00] [INFO] [test.go:3] ["at end"]`,
		`[2026/05/26 10:01:01.000 +08:00] [INFO] [test.go:4] ["after"]`,
	}, "\n")+"\n"), 0644))

	s := &LogScraper{
		Paths: []string{logPath},
		Types: map[string]bool{LogTypeStd: true},
		Start: start,
		End:   end,
	}
	result := &Sample{}
	require.NoError(t, s.Scrap(result))
	require.Len(t, result.Log, 1)

	var filteredPath string
	for p := range result.Log {
		filteredPath = p
	}
	require.NotEqual(t, logPath, filteredPath)
	require.Equal(t, logPath, result.LogTargets[filteredPath])
	require.Equal(t, LogTypeStd, result.LogTypes[filteredPath])

	filtered, err := os.ReadFile(filteredPath)
	require.NoError(t, err)
	filteredContent := string(filtered)
	require.NotContains(t, filteredContent, "before")
	require.Contains(t, filteredContent, "at start")
	require.Contains(t, filteredContent, "at end")
	require.NotContains(t, filteredContent, "after")
}

func mustParseLogTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse("2006/01/02 15:04:05.000 -07:00", s)
	require.NoError(t, err)
	return ts
}
