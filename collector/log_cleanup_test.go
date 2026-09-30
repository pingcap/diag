package collector

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/stretchr/testify/require"
)

// Only interpret the two directory operations; no SSH or shell is executed.
type trimExecutor struct {
	mu        sync.Mutex
	commands  []string
	removeErr error
}

func (e *trimExecutor) Execute(ctx context.Context, cmd string, sudo bool, timeout ...time.Duration) ([]byte, []byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.commands = append(e.commands, cmd)
	if strings.HasPrefix(cmd, "mkdir -m 700 -- '") {
		return nil, nil, os.Mkdir(strings.TrimSuffix(strings.TrimPrefix(cmd, "mkdir -m 700 -- '"), "'"), 0700)
	}
	if strings.HasPrefix(cmd, "rm -rf -- '") {
		if e.removeErr != nil {
			return nil, nil, e.removeErr
		}
		return nil, nil, os.RemoveAll(strings.TrimSuffix(strings.TrimPrefix(cmd, "rm -rf -- '"), "'"))
	}
	return nil, nil, errors.New("unexpected command: " + cmd)
}
func (e *trimExecutor) Transfer(context.Context, string, string, bool, int, bool) error {
	return errors.New("download failed")
}
func cleanupFixture(t *testing.T) (*LogCollectOptions, context.Context, *trimExecutor) {
	t.Helper()
	c := &LogCollectOptions{}
	c.trimPath = filepath.Join(t.TempDir(), filepath.Base(c.trimDir()))
	ctx := ctxt.New(context.Background(), 1, logprinter.NewLogger(""))
	exec := &trimExecutor{}
	ctxt.GetInner(ctx).SetExecutor("host", exec)
	return c, ctx, exec
}

func TestTrimDirectoryIsolation(t *testing.T) {
	first, ctx, exec := cleanupFixture(t)
	second := &LogCollectOptions{trimPath: filepath.Join(filepath.Dir(first.trimPath), "another-run")}
	require.NoError(t, first.prepareTrimDir("host").Execute(ctx))
	require.NoError(t, second.prepareTrimDir("host").Execute(ctx))
	fi, err := os.Stat(first.trimPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0700), fi.Mode().Perm())
	first.Close()
	require.NoDirExists(t, first.trimPath)
	require.DirExists(t, second.trimPath)
	require.NotContains(t, strings.Join(exec.commands, "\n"), "rm -rf -- '/tmp/diag-trimmed'")
	count := len(exec.commands)
	first.Close()
	require.Len(t, exec.commands, count, "Close is idempotent")
	second.Close()
	require.NoDirExists(t, second.trimPath)
}

func TestTrimCleanupWithoutCollect(t *testing.T) {
	for _, reason := range []string{"confirmation cancelled", "later prepare failed"} {
		t.Run(reason, func(t *testing.T) {
			c, ctx, _ := cleanupFixture(t)
			// The manager registers Close before Prepare; these exits never call Collect.
			err := func() error {
				defer c.Close()
				require.NoError(t, c.prepareTrimDir("host").Execute(ctx))
				require.NoError(t, os.WriteFile(filepath.Join(c.trimPath, "rocksdb.info"), []byte("log"), 0600))
				return errors.New(reason)
			}()
			require.EqualError(t, err, reason)
			require.NoDirExists(t, c.trimPath)
		})
	}
}

func TestTrimCleanupAfterDownload(t *testing.T) {
	for _, fail := range []bool{false, true} {
		c, ctx, _ := cleanupFixture(t)
		require.NoError(t, c.prepareTrimDir("host").Execute(ctx))
		download := task.NewFunc("download", func(ctx context.Context) error {
			require.DirExists(t, c.trimPath, "source survives until download")
			if fail {
				exec, _ := ctxt.GetInner(ctx).GetExecutor("host")
				return exec.Transfer(ctx, "source", "dest", true, 0, false)
			}
			return nil
		})
		err := c.runLogDownload(ctx, download)
		if fail {
			require.EqualError(t, err, "download failed")
		} else {
			require.NoError(t, err)
		}
		require.NoDirExists(t, c.trimPath)
	}
}

func TestFailedTrimDirectoryCreationDoesNotRemoveExistingDirectory(t *testing.T) {
	c, ctx, exec := cleanupFixture(t)
	require.NoError(t, os.Mkdir(c.trimPath, 0700))
	require.Error(t, c.prepareTrimDir("host").Execute(ctx))
	c.Close()
	require.DirExists(t, c.trimPath)
	require.Len(t, exec.commands, 1, "never register cleanup for a directory we did not create")
}

func TestTrimCleanupContinuesAfterHostFailure(t *testing.T) {
	c, ctx, failed := cleanupFixture(t)
	require.NoError(t, c.prepareTrimDir("host").Execute(ctx))
	failed.removeErr = errors.New("host disconnected")
	other := &trimExecutor{}
	ctxt.GetInner(ctx).SetExecutor("other", other)
	// Different hosts may have the same remote path. Remove the local fixture to
	// simulate a separate filesystem for the second host.
	require.NoError(t, os.Remove(c.trimPath))
	require.NoError(t, c.prepareTrimDir("other").Execute(ctx))
	c.Close()
	require.Len(t, failed.commands, 2)
	require.Len(t, other.commands, 2, "one host's failure must not skip another host")
}
