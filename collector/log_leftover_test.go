// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pingcap/tiup/pkg/cluster/task"
	"github.com/pingcap/tiup/pkg/tui"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// what a leftover is
// ---------------------------------------------------------------------------

// TestTrimDirRuleDecidesWhatMayBeRemoved pins the rule every removal follows: a
// directory directly below trimDirRoot, named the way trimDir names the ones it
// creates. A rule that is too wide removes data this collector does not own - the
// tools directory and the root itself live right next to the trimmed logs - and a
// rule that is too narrow removes nothing at all, so both directions are checked.
func TestTrimDirRuleDecidesWhatMayBeRemoved(t *testing.T) {
	assert := require.New(t)
	root := trimDirRoot()
	runID := trimDirPrefix + uuid.NewString()

	// the rule has to accept what trimDir builds, or no leftover would ever be
	// removed; this is also what keeps trimDir and the rule from drifting apart
	assert.True(isTrimDirName(filepath.Base((&LogCollectOptions{}).trimDir())))
	assert.NoError(assertRemovableTrimDir(filepath.Join(root, runID)))

	for _, dir := range []string{
		"",
		root, // the directory holding the trimmed logs is never removed
		// the tools directory of every collector, which lives in the same root
		task.CheckToolsPathDir,
		filepath.Join(root, "other"),
		filepath.Join(root, "trimmed"),
		filepath.Join(root, trimDirPrefix),              // the prefix without a run id
		filepath.Join(root, trimDirPrefix+"not-a-uuid"), // not a name trimDir can produce
		filepath.Join(root, runID+"-more"),              // a run id with something appended
		filepath.Join(root, runID, "nested"),            // one level too deep
		filepath.Join(root, "sub", runID),               // not directly below the root
		filepath.Join(root, "..", runID),                // a path that only cleans into the root
		runID,                                           // relative
		filepath.Join(root, runID) + "/",                // not clean
	} {
		assert.Error(assertRemovableTrimDir(dir), "must not be removable: %q", dir)
	}
}

// TestOwnedTrimmedLogsKeepsForeignEntriesApart checks the split deciding what is
// offered to the user: only the directories this version of the trimming created
// may be removed, anything else the probe matched is kept out of it.
func TestOwnedTrimmedLogsKeepsForeignEntriesApart(t *testing.T) {
	assert := require.New(t)
	ours := filepath.Join(trimDirRoot(), trimDirPrefix+uuid.NewString())
	foreign := filepath.Join(trimDirRoot(), trimDirPrefix+"not-a-uuid")

	owned, others := ownedTrimmedLogs([]leftoverTrimmedLog{
		{path: ours, size: 1},
		{path: foreign, size: 2},
	})

	assert.Equal([]leftoverTrimmedLog{{path: ours, size: 1}}, owned)
	assert.Equal([]leftoverTrimmedLog{{path: foreign, size: 2}}, others)
}

// ---------------------------------------------------------------------------
// what the hosts are asked
// ---------------------------------------------------------------------------

// executorPrefix is what every cluster executor puts in front of the command it
// runs, remote or local (tiup pkg/cluster/executor/ssh.go and local.go). It ends
// with a space and carries no separator, so whatever follows it has to be a
// single simple command: a bare `for ... done` is a syntax error on the host,
// which only a real host - or a test that goes through the same prefix - shows.
const executorPrefix = "export LANG=C; PATH=$PATH:/bin:/sbin:/usr/bin:/usr/sbin "

// runLikeAnExecutor runs cmd the way a target host receives it.
func runLikeAnExecutor(t *testing.T, cmd string) string {
	t.Helper()
	out, err := exec.Command("sh", "-c", executorPrefix+cmd).Output()
	require.NoError(t, err, "output: %q", out)
	return string(out)
}

// TestLeftoverProbeIsASingleSimpleCommand pins the shape the probe has to have.
// The executors prepend an assignment without a separator, so a compound command
// makes the host fail with "syntax error near unexpected token `do'". This is
// checked by the shell itself rather than by comparing strings.
func TestLeftoverProbeIsASingleSimpleCommand(t *testing.T) {
	require := require.New(t)

	cmd := executorPrefix + leftoverTrimmedLogsCmdIn(t.TempDir())
	require.NoError(exec.Command("sh", "-n", "-c", cmd).Run(),
		"the probe must be valid in one command: %s", cmd)

	// and the shape that is not: a loop sent as it is
	compound := executorPrefix + `for d in /tmp/x*; do echo "$d"; done`
	require.Error(exec.Command("sh", "-n", "-c", compound).Run(),
		"a compound command can not follow the prefix, so the probe must not be one")
}

// TestLeftoverProbeReportsCandidatesWithoutTouchingThem runs the command the
// hosts receive against a directory shaped the way the real one is: it has to
// report the directories that could hold trimmed logs with their size, leave out
// files that only match by name, and never remove or create anything. The
// ownership decision is deliberately not part of the command - it is taken once,
// in Go, by the same rule the removal uses.
func TestLeftoverProbeReportsCandidatesWithoutTouchingThem(t *testing.T) {
	assert := require.New(t)
	root := t.TempDir()

	ours := filepath.Join(root, trimDirPrefix+uuid.NewString())
	assert.NoError(os.MkdirAll(filepath.Join(ours, "data", "db", "data"), 0o700))
	assert.NoError(os.WriteFile(filepath.Join(ours, "data", "db", "data", "rocksdb.info"),
		[]byte(strings.Repeat("x", 4096)), 0o600))

	lookalike := filepath.Join(root, trimDirPrefix+"not-a-uuid")
	assert.NoError(os.MkdirAll(lookalike, 0o700))

	// a file matching the name pattern is not a directory of trimmed logs
	fileLike := filepath.Join(root, trimDirPrefix+uuid.NewString())
	assert.NoError(os.WriteFile(fileLike, []byte("junk"), 0o600))

	other := filepath.Join(root, "other")
	assert.NoError(os.MkdirAll(other, 0o700))

	out := runLikeAnExecutor(t, leftoverTrimmedLogsCmdIn(root))

	reported := map[string]int64{}
	for _, l := range parseLeftoverTrimmedLogs(out) {
		reported[l.path] = l.size
	}
	assert.Len(reported, 2, "found %v in %q", reported, out)
	assert.Contains(reported, ours)
	assert.Contains(reported, lookalike, "the command reports candidates, the rule decides")
	assert.NotContains(reported, fileLike)
	assert.NotContains(reported, other)
	assert.NotContains(reported, root)
	assert.GreaterOrEqual(reported[ours], int64(1024), "the size of the directory is reported")

	// nothing was removed and nothing was created
	entries, err := os.ReadDir(root)
	assert.NoError(err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.ElementsMatch([]string{
		filepath.Base(ours), filepath.Base(lookalike), filepath.Base(fileLike), filepath.Base(other),
	}, names)
}

// TestLeftoverProbeWithoutLeftoversIsSilentAndSuccessful runs the probe in an
// empty directory, and in one that does not exist at all. A host without
// leftovers has to report nothing and still exit successfully, otherwise every
// collection would fail on a clean host.
func TestLeftoverProbeWithoutLeftoversIsSilentAndSuccessful(t *testing.T) {
	assert := require.New(t)

	for _, root := range []string{t.TempDir(), filepath.Join(t.TempDir(), "does-not-exist")} {
		out := runLikeAnExecutor(t, leftoverTrimmedLogsCmdIn(root))
		assert.Empty(parseLeftoverTrimmedLogs(out), "root %s, output: %q", root, out)
	}
}

// TestParseLeftoverTrimmedLogs checks the reading of the probe output, including
// the shapes a host produces when du is missing and when a name contains spaces.
func TestParseLeftoverTrimmedLogs(t *testing.T) {
	assert := require.New(t)

	assert.Empty(parseLeftoverTrimmedLogs(""))
	assert.Empty(parseLeftoverTrimmedLogs("/tmp/diag-trimmed-x\n")) // no size column
	assert.Empty(parseLeftoverTrimmedLogs("not a path line"))
	assert.Empty(parseLeftoverTrimmedLogs("\t12"))

	assert.Equal([]leftoverTrimmedLog{
		// the size is reported in kibibytes and read in bytes
		{path: "/tmp/diag-trimmed-a", size: 2048},
		// a host where du failed still reports the directory, with no size
		{path: "/tmp/diag-trimmed-b", size: 0},
		{path: "/tmp/a dir with spaces/diag-trimmed-c", size: 1024},
		// a negative size is not trusted
		{path: "/tmp/diag-trimmed-d", size: 0},
	}, parseLeftoverTrimmedLogs(strings.Join([]string{
		"/tmp/diag-trimmed-a\t2",
		"/tmp/diag-trimmed-b\t",
		"/tmp/a dir with spaces/diag-trimmed-c\t1",
		"/tmp/diag-trimmed-d\t-1",
	}, "\n")))
}

// ---------------------------------------------------------------------------
// removing them
// ---------------------------------------------------------------------------

// TestLeftoverRemovalRemovesExactlyTheCheckedDirectories checks the steps that
// delete: one per host, carrying the leftovers of that host, and never the
// directory holding them or the tools directory next to them. It also checks that
// a leftover failing the rule stops the removal instead of being handed to rm.
func TestLeftoverRemovalRemovesExactlyTheCheckedDirectories(t *testing.T) {
	assert := require.New(t)
	topo := testTiUPTopology(t)
	m := testManager()
	opt := testLogOptions(collectLog{Rocksdb: true})
	const host = "127.0.0.1"

	first := filepath.Join(trimDirRoot(), trimDirPrefix+uuid.NewString())
	second := filepath.Join(trimDirRoot(), trimDirPrefix+uuid.NewString())
	steps, err := opt.buildLeftoverRemovalSteps(m, topo, []string{host}, map[string][]leftoverTrimmedLog{
		host: {{path: first, size: 1024}, {path: second, size: 2048}},
	})
	assert.NoError(err)

	rendered := renderSteps(steps)
	// the whole rendered removal: the two leftovers of the host, nothing else.
	// The directory holding them and the tools directory next to them are the
	// accidents this pins down.
	assert.Equal([]string{
		"directories='" + first + "'",
		"directories='" + second + "'",
	}, removedDirectories(rendered), "rendered:\n%s", rendered)

	// a host without leftovers gets no step at all
	steps, err = opt.buildLeftoverRemovalSteps(m, topo, []string{host}, nil)
	assert.NoError(err)
	assert.Empty(steps)

	// an unchecked path never reaches a step, whichever one it is
	for _, dir := range []string{
		trimDirRoot(),
		task.CheckToolsPathDir,
		filepath.Join(trimDirRoot(), trimDirPrefix+"not-a-uuid"),
	} {
		steps, err := opt.buildLeftoverRemovalSteps(m, topo, []string{host}, map[string][]leftoverTrimmedLog{
			host: {{path: dir}},
		})
		assert.Error(err, "must not build a removal for %q", dir)
		assert.Empty(steps)
	}
}

// removedDirectories extracts what the rendered removal steps delete, so that a
// test can compare the whole set instead of looking for a path it hopes to find.
func removedDirectories(rendered string) []string {
	return regexp.MustCompile(`directories='[^']*'`).FindAllString(rendered, -1)
}

// ---------------------------------------------------------------------------
// who is asked about leftovers
// ---------------------------------------------------------------------------

// tiupMultiHostTopology spreads the components over several hosts, so that the
// hosts a collection visits - and therefore the hosts asked about leftovers - can
// be told apart from the ones it does not.
const tiupMultiHostTopology = `global:
  user: ubuntu
  ssh_port: 22
  deploy_dir: /data/db/deploy
  data_dir: /data/db/data
pd_servers:
  - host: 10.0.0.1
    client_port: 2379
tikv_servers:
  - host: 10.0.0.1
    port: 20160
    status_port: 20180
  - host: 10.0.0.1
    port: 20161
    status_port: 20181
tidb_servers:
  - host: 10.0.0.2
    port: 4000
    status_port: 10080
monitoring_servers:
  - host: 10.0.0.1
    port: 9090
grafana_servers:
  - host: 10.0.0.3
`

// TestLogHostsFollowsTheFilters checks where leftovers are looked for: on every
// host that carries logs this collection takes, once each, and on no other host.
// A host the collection does not visit must not be asked, and a host carrying
// several instances - the usual case for TiKV - must not be asked twice.
func TestLogHostsFollowsTheFilters(t *testing.T) {
	assert := require.New(t)
	topo := testTopology(t, tiupMultiHostTopology)

	opt := testLogOptions(collectLog{Rocksdb: true})
	// grafana on 10.0.0.3 carries no logs this collector takes
	assert.Equal([]string{"10.0.0.1", "10.0.0.2"}, opt.logHosts(topo))

	// --node selects instances by id, host:port
	opt.opt.Nodes = []string{"10.0.0.2:4000"}
	assert.Equal([]string{"10.0.0.2"}, opt.logHosts(topo))

	// only grafana lives on that host, and grafana is not collected
	opt.opt.Nodes = []string{"10.0.0.3:3000"}
	assert.Empty(opt.logHosts(topo))

	opt.opt.Nodes = []string{"10.0.0.9:4000"}
	assert.Empty(opt.logHosts(topo))

	opt.opt.Nodes = nil
	opt.opt.Roles = []string{"tikv"}
	assert.Equal([]string{"10.0.0.1"}, opt.logHosts(topo))

	opt.opt.Roles = []string{"grafana"}
	assert.Empty(opt.logHosts(topo))
}

// TestLeftoverRemovalFollowsTheAnswer covers who decides: the flag that asks for
// the removal and -y, which answers yes to every question, remove without asking;
// an interactive run asks and follows the answer. Nothing is removed behind the
// user's back, so a declined question keeps the leftovers.
func TestLeftoverRemovalFollowsTheAnswer(t *testing.T) {
	asked := 0
	answer := false
	t.Cleanup(func() { confirmRemoveLeftover = tui.PromptForConfirmYes })
	confirmRemoveLeftover = func(string, ...interface{}) (bool, string) {
		asked++
		return answer, ""
	}

	for _, tc := range []struct {
		name      string
		clean     bool
		skip      bool
		answer    bool
		want      bool
		wantAsked int
	}{
		{name: "the flag removes without asking", clean: true, want: true},
		{name: "-y removes without asking", skip: true, want: true},
		{name: "an interactive run removes when the user says yes", answer: true, want: true, wantAsked: 1},
		{name: "an interactive run keeps them when the user says no", want: false, wantAsked: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert := require.New(t)
			asked = 0
			answer = tc.answer
			opt := &LogCollectOptions{cleanLeftover: tc.clean, skipConfirm: tc.skip}

			assert.Equal(tc.want, opt.confirmLeftoverTrimmedRemoval())
			assert.Equal(tc.wantAsked, asked)
		})
	}
}

// TestLeftoverDescriptionNamesWhatIsRemoved checks what the user is shown before
// answering: every directory with its size, and a total over the hosts.
func TestLeftoverDescriptionNamesWhatIsRemoved(t *testing.T) {
	assert := require.New(t)
	first := filepath.Join(trimDirRoot(), trimDirPrefix+uuid.NewString())
	second := filepath.Join(trimDirRoot(), trimDirPrefix+uuid.NewString())

	desc, total := describeLeftoverTrimmedLogs(
		[]string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
		map[string][]leftoverTrimmedLog{
			"10.0.0.1": {{path: first, size: 1024 * 1024}},
			"10.0.0.2": {{path: second, size: 2 * 1024 * 1024}},
		})

	assert.Equal(int64(3*1024*1024), total)
	assert.Contains(desc, "10.0.0.1: "+first)
	assert.Contains(desc, "10.0.0.2: "+second)
	assert.NotContains(desc, "10.0.0.3", "a host without leftovers is not mentioned")
}
