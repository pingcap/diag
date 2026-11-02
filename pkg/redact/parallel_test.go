// Copyright 2026 PingCAP, Inc.
// Licensed under the Apache License, Version 2.0.

package redact

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParallelCopyAndRewrite(t *testing.T) {
	root := t.TempDir()
	input, inplace := filepath.Join(root, "input"), filepath.Join(root, "inplace")
	for i := 0; i < 32; i++ {
		writeFixture(t, input, fmt.Sprintf("host-%02d/tidb.log", i), fmt.Sprintf("peer=10.1.1.%d\n", i+1))
	}
	require.NoError(t, os.CopyFS(inplace, os.DirFS(input)))
	serial, err := DiscoverWithWorkers(input, nil, 1)
	require.NoError(t, err)
	parallel, err := DiscoverWithWorkers(input, func(Progress) {}, 4)
	require.NoError(t, err)
	_, a, err := serial.Copy(input, filepath.Join(root, "serial"))
	require.NoError(t, err)
	_, b, err := parallel.Copy(input, filepath.Join(root, "parallel"))
	require.NoError(t, err)
	require.Equal(t, a, b)
	rewrite, err := DiscoverWithWorkers(inplace, func(Progress) {}, 4)
	require.NoError(t, err)
	_, c, err := rewrite.Rewrite(inplace)
	require.NoError(t, err)
	require.Equal(t, a, c)
	for i := 0; i < 32; i++ {
		name := fmt.Sprintf("host-%02d/tidb.log", i)
		want, err := os.ReadFile(filepath.Join(root, "serial", name))
		require.NoError(t, err)
		for _, dir := range []string{"parallel", "inplace"} {
			got, err := os.ReadFile(filepath.Join(root, dir, name))
			require.NoError(t, err)
			require.Equal(t, want, got)
		}
	}
	require.NoError(t, CheckInPlaceComplete(inplace))
}

func TestParallelFailuresAndLimits(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	for i := 0; i < 12; i++ {
		writeFixture(t, input, fmt.Sprintf("%02d.log", i), strings.Repeat("10.1.1.1 ", 100))
	}
	r, err := New([]string{"10.1.1.1"})
	require.NoError(t, err)
	require.Error(t, r.SetWorkers(0))
	require.Error(t, r.SetWorkers(257))
	require.NoError(t, r.SetWorkers(4))
	plan, err := r.prepareRewrite(input)
	require.NoError(t, err)
	writeFixture(t, input, "05.log", "changed after preflight")
	require.ErrorContains(t, r.applyRewrite(input, plan), "changed after preflight")
	require.ErrorContains(t, CheckInPlaceComplete(input), "incomplete")
	entries, err := os.ReadDir(input)
	require.NoError(t, err)
	for _, entry := range entries {
		require.False(t, strings.HasPrefix(entry.Name(), ".diag-redact-file-"))
	}

	limit, err := New([]string{"10.1.1.1"})
	require.NoError(t, err)
	limit.remaining = 10
	require.NoError(t, limit.SetWorkers(4))
	pool := limit.newPool()
	for i := 0; i < 4; i++ {
		if pool.submit(func(w *Redactor) error { return w.plain(strings.NewReader("abcdef"), io.Discard) }) != nil {
			break
		}
	}
	require.ErrorContains(t, pool.finish(limit), "expanded data limit")

	limit.shared = &sharedState{remaining: maxExpandedBytes, entries: maxEntries - 1}
	pool = limit.newPool()
	for i := 0; i < 2; i++ {
		if pool.submit(func(w *Redactor) error { return w.entry() }) != nil {
			break
		}
	}
	require.ErrorContains(t, pool.finish(limit), "file count limit")
}

func TestParallelCopyFailureDoesNotPublish(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	for i := 0; i < 12; i++ {
		writeFixture(t, input, fmt.Sprintf("%02d.log", i), "peer=10.1.1.1\n")
	}
	require.NoError(t, os.WriteFile(filepath.Join(input, "05.log"), []byte{0, 1, 2}, 0600))
	r, err := New([]string{"10.1.1.1"})
	require.NoError(t, err)
	require.NoError(t, r.SetWorkers(4))
	_, _, err = r.Copy(input, filepath.Join(root, "output"))
	require.Error(t, err)
	_, err = os.Stat(filepath.Join(root, "output"))
	require.True(t, os.IsNotExist(err))
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}
