package iterator

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pingcap/diag/collector/log/parser"
	"github.com/stretchr/testify/require"
)

func TestLogIteratorIncludesEntryAtBeginTime(t *testing.T) {
	root := t.TempDir()
	logDir := filepath.Join(root, "127.0.0.1", "tidb-4000")
	require.NoError(t, os.MkdirAll(logDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(logDir, "tidb.log"), []byte(
		`[2026/05/26 09:59:59.000 +08:00] [INFO] [test.go:1] ["before"]`+"\n"+
			`[2026/05/26 10:00:00.000 +08:00] [INFO] [test.go:2] ["at start"]`+"\n"+
			`[2026/05/26 10:00:01.000 +08:00] [INFO] [test.go:3] ["inside"]`+"\n",
	), 0644))

	begin := mustParseIteratorTime(t, "2026/05/26 10:00:00.000 +08:00")
	end := mustParseIteratorTime(t, "2026/05/26 10:00:01.000 +08:00")
	iter, err := New(parser.NewFileWrapper(root, "127.0.0.1", "tidb-4000", "tidb.log"), begin, end)
	require.NoError(t, err)
	defer iter.Close()

	first, err := iter.Next()
	require.NoError(t, err)
	require.Contains(t, string(first.GetContent()), "at start")

	second, err := iter.Next()
	require.NoError(t, err)
	require.Contains(t, string(second.GetContent()), "inside")

	_, err = iter.Next()
	require.ErrorIs(t, err, io.EOF)
}

func mustParseIteratorTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse("2006/01/02 15:04:05.000 -07:00", s)
	require.NoError(t, err)
	return ts
}
