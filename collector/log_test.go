package collector

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pingcap/diag/scraper"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/stretchr/testify/require"
)

// TestCollectScrapedStats covers the step between the scraper and the download:
// the sample the scraper printed has to reach fileStats. Everything the scraper
// reports now goes through this one method, so a merge that is missing, that
// parses nothing, or that overwrites the previous scrape of the same host is
// caught here instead of silently shipping the wrong file list.
func TestCollectScrapedStats(t *testing.T) {
	assert := require.New(t)
	const host = "127.0.0.1"

	ctx := ctxt.New(context.Background(), 1, logprinter.NewLogger(""))
	scrape := func(stdout string) {
		ctxt.GetInner(ctx).SetOutputs(host, []byte(stdout), nil)
	}
	opt := &LogCollectOptions{fileStats: map[string][]CollectStat{}}

	scrape(`{"log_files":{"/data/db/data/tikv-20160/rocksdb.info":123},` +
		`"config_files":{"/data/db/deploy/tikv-20160/conf/tikv.toml":7},` +
		`"prometheus_data":{"/data/db/data/prom/tmp/blocks":9}}`)
	assert.NoError(opt.collectScrapedStats(ctx, host))
	assert.Equal([]CollectStat{
		{Target: "/data/db/deploy/tikv-20160/conf/tikv.toml", Size: 7},
		{Target: "/data/db/data/tikv-20160/rocksdb.info", Size: 123},
		{Target: "/data/db/data/prom/tmp/blocks", Size: 9},
	}, opt.fileStats[host])

	// a second scrape of the same host (another TiKV instance) accumulates
	scrape(`{"log_files":{"/data/db/data/tikv-20161/rocksdb.info":456}}`)
	assert.NoError(opt.collectScrapedStats(ctx, host))
	assert.Len(opt.fileStats[host], 4, "the previous scrape must not be overwritten")

	// nothing printed means nothing to merge, and must not fail
	scrape("")
	assert.NoError(opt.collectScrapedStats(ctx, host))

	// anything else the scraper could print is an error, not a silent skip
	scrape("not json")
	assert.Error(opt.collectScrapedStats(ctx, host))
}

// ---------------------------------------------------------------------------
// the lifetime of what the collector writes on the target hosts
// ---------------------------------------------------------------------------

// TestTrimDirOutlivesOtherCollectors guards the lifetime of the trimmed copies:
// they are written during Prepare and downloaded during Collect, so they must
// not live where another collector deletes while it collects. system and TSDB
// (raw monitor mode) remove task.CheckToolsPathDir during their own Collect and
// are registered before the log collector, so a trim directory below it would be
// gone by the time the download starts and the rocksdb logs would silently miss
// from the package.
func TestTrimDirOutlivesOtherCollectors(t *testing.T) {
	c := &LogCollectOptions{}
	assert := require.New(t)

	assert.NotEqual(task.CheckToolsPathDir, c.trimDir())
	assert.False(isWithin(task.CheckToolsPathDir, c.trimDir()),
		"the trim dir %q must not live inside %q, which other collectors remove before the log collector downloads its files",
		c.trimDir(), task.CheckToolsPathDir)
}

func TestTrimDirIsUniqueAndStable(t *testing.T) {
	first, second := &LogCollectOptions{}, &LogCollectOptions{}
	dir := first.trimDir()
	require.Equal(t, dir, first.trimDir())
	require.NotEqual(t, dir, second.trimDir())
}

// TestPrepareCollectsNothingWithoutAType pins the early return of Prepare
// itself, not just the predicate behind it: selecting nothing, or only log.ops
// (which the audit log collector owns), must not walk the topology, must not
// build any task and must not fail.
func TestPrepareCollectsNothingWithoutAType(t *testing.T) {
	assert := require.New(t)

	for _, c := range []collectLog{{}, {Ops: true}} {
		opt := &LogCollectOptions{collector: c}
		// the cluster is nil on purpose: returning before it is dereferenced is
		// part of what this asserts
		stats, err := opt.Prepare(&Manager{mode: CollectModeTiUP}, nil)
		assert.NoError(err)
		assert.Nil(stats, "collector %+v", c)
	}

	// an unknown collection mode is a no-op as well, and must not panic
	opt := &LogCollectOptions{collector: collectLog{Rocksdb: true}}
	stats, err := opt.Prepare(&Manager{mode: "not-a-mode"}, nil)
	assert.NoError(err)
	assert.Nil(stats)
}

// isWithin reports whether child is inside parent.
func isWithin(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != "." && rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ---------------------------------------------------------------------------
// what the collectors ask the scraper to do
// ---------------------------------------------------------------------------

// noEmptyFlagValue fails when a command ends with a flag that got no value: that
// shape made the scraper exit with "flag needs an argument: --logtype" and
// abort the whole collection, so it must never be generated again.
func noEmptyFlagValue(t *testing.T, cmd string) {
	t.Helper()
	assert := require.New(t)
	assert.NotEmpty(cmd)
	assert.NotRegexp(`--[a-zA-Z-]+\s*$`, cmd, "command ends with a flag without a value: %s", cmd)
}

// TestScraperCommandShapes is the compact matrix of the two commands: the
// rocksdb one is the only one that trims, and neither may carry a flag without
// a value. The steps the collector really builds are asserted in
// log_tiup_test.go, against a topology.
func TestScraperCommandShapes(t *testing.T) {
	assert := require.New(t)

	rocks := rocksdbScraperCmd("/data/db/data/tikv-20160", "b", "e", "/tmp/diag-trimmed")
	noEmptyFlagValue(t, rocks)
	assert.True(strings.HasPrefix(rocks, scraperPath()), rocks)
	assert.Contains(rocks, "--log '/data/db/data/tikv-20160/*'")
	assert.Contains(rocks, "--logtype rocksdb --trim --trim-dir '/tmp/diag-trimmed'")

	// trimming is scoped to the rocksdb logs: cutting the component logs at
	// both ends would change what every existing user gets, and would empty the
	// stderr logs which are collected regardless of the time range on purpose
	generic, ok := genericScraperCmd([]string{"/data/pd-2379/log/*"}, "b", "e",
		[]string{scraper.LogTypeStd, scraper.LogTypeSlow})
	assert.True(ok)
	assert.Contains(generic, "--log '/data/pd-2379/log/*'")
	assert.Contains(generic, "--logtype std,slow")
	assert.NotContains(generic, "--trim", "component logs are collected whole on purpose")
	noEmptyFlagValue(t, generic)
}

func TestGenericScraperCmdNeedsAtLeastOneType(t *testing.T) {
	assert := require.New(t)

	// no type of the component log directories requested: no command at all,
	// the step has to be skipped instead of being run with a bare --logtype
	cmd, ok := genericScraperCmd([]string{"/data/pd-2379/log/*"}, "b", "e", nil)
	assert.False(ok)
	assert.Empty(cmd)

	cmd, ok = genericScraperCmd([]string{"/data/pd-2379/log/*"}, "b", "e", []string{scraper.LogTypeStd})
	assert.True(ok)
	assert.Contains(cmd, "--log '/data/pd-2379/log/*'")
	assert.Contains(cmd, "--logtype std")
	noEmptyFlagValue(t, cmd)
}

// TestScraperCommandsNeverCarryAnEmptyFlagValue walks every combination of the
// collectLog switches: this is the class of the "flag needs an argument"
// failure, which must not come back whatever the user selects.
func TestScraperCommandsNeverCarryAnEmptyFlagValue(t *testing.T) {
	assert := require.New(t)
	paths := []string{"/data/pd-2379/log/*", "/data/tikv-20160/log/*"}

	for i := 0; i < 16; i++ {
		opt := &LogCollectOptions{collector: collectLog{
			Std:     i&1 != 0,
			Slow:    i&2 != 0,
			Unknown: i&4 != 0,
			Ops:     i&8 != 0,
		}}
		cmd, ok := genericScraperCmd(paths, "b", "e", opt.logTypesToScrap())
		if !ok {
			continue
		}
		noEmptyFlagValue(t, cmd)
		assert.Regexp(`--logtype \S+`, cmd)
	}
}

// TestNeedsCollectMatchesTheSelectedTypes pins the semantics of the early
// return: everything the user can ask for is collected, log.ops is not (the
// audit log collector owns it), and the predicate is derived from
// logTypesToScrap so the two can not drift apart again - keeping them in sync
// by hand is how log.unknown once ended up collected by nobody.
func TestNeedsCollectMatchesTheSelectedTypes(t *testing.T) {
	assert := require.New(t)

	for _, std := range []bool{false, true} {
		for _, slow := range []bool{false, true} {
			for _, unknown := range []bool{false, true} {
				for _, rocksdb := range []bool{false, true} {
					for _, ops := range []bool{false, true} {
						opt := &LogCollectOptions{collector: collectLog{
							Std: std, Slow: slow, Unknown: unknown, Rocksdb: rocksdb, Ops: ops,
						}}
						assert.Equal(std || slow || unknown || rocksdb, opt.needsCollect(),
							"std=%v slow=%v unknown=%v rocksdb=%v ops=%v", std, slow, unknown, rocksdb, ops)
					}
				}
			}
		}
	}
}

func TestPathInPackage(t *testing.T) {
	c := &LogCollectOptions{}
	assert := require.New(t)

	// a trimmed rocksdb log keeps its place in the package instead of showing
	// the temporary directory it was written to on the remote host
	assert.Equal(
		filepath.Join("/tmp/result", "127.0.0.1", "data/tikv-20160/rocksdb.info"),
		c.pathInPackage("/tmp/result", "127.0.0.1",
			filepath.Join(c.trimDir(), "data/tikv-20160/rocksdb.info")),
	)
	// any other file keeps its absolute path, as before
	assert.Equal(
		filepath.Join("/tmp/result", "127.0.0.1", "/data/tikv-20160/log/tikv.log"),
		c.pathInPackage("/tmp/result", "127.0.0.1", "/data/tikv-20160/log/tikv.log"),
	)
	// the trim directory itself and its parent are not treated as trimmed files
	assert.Equal(
		filepath.Join("/tmp/result", "127.0.0.1", c.trimDir()),
		c.pathInPackage("/tmp/result", "127.0.0.1", c.trimDir()),
	)
	assert.Equal(
		filepath.Join("/tmp/result", "127.0.0.1", "/tmp/tiup"),
		c.pathInPackage("/tmp/result", "127.0.0.1", "/tmp/tiup"),
	)
	// only the real trim dir is stripped: a file below the shared tools dir
	// keeps its path, so a wrong trim location shows up in the package instead
	// of being quietly rewritten
	assert.Equal(
		filepath.Join("/tmp/result", "127.0.0.1",
			filepath.Join(task.CheckToolsPathDir, "trimmed", "rocksdb.info")),
		c.pathInPackage("/tmp/result", "127.0.0.1",
			filepath.Join(task.CheckToolsPathDir, "trimmed", "rocksdb.info")),
	)
}

func TestLogTypesToScrap(t *testing.T) {
	assert := require.New(t)

	cases := []struct {
		name      string
		collector collectLog
		expected  []string
	}{
		{
			name:      "std only",
			collector: collectLog{Std: true},
			expected:  []string{scraper.LogTypeStd},
		},
		{
			name:      "slow only",
			collector: collectLog{Slow: true},
			expected:  []string{scraper.LogTypeSlow},
		},
		{
			name:      "unknown only",
			collector: collectLog{Unknown: true},
			expected:  []string{scraper.LogTypeUnknown},
		},
		{
			name:      "std, slow and unknown",
			collector: collectLog{Std: true, Slow: true, Unknown: true},
			expected:  []string{scraper.LogTypeStd, scraper.LogTypeSlow, scraper.LogTypeUnknown},
		},
		{
			// rocksdb logs are scrapped from the TiKV data directory by a
			// dedicated step, so nothing is left for the generic scraper step
			// and it must not be built with an empty --logtype value
			name:      "rocksdb only",
			collector: collectLog{Rocksdb: true},
			expected:  nil,
		},
		{
			name:      "nothing selected",
			collector: collectLog{},
			expected:  nil,
		},
	}

	for _, cs := range cases {
		opt := &LogCollectOptions{collector: cs.collector}
		assert.Equal(cs.expected, opt.logTypesToScrap(), cs.name)
	}
}

func TestMergeFileStats(t *testing.T) {
	assert := require.New(t)

	// several TiKV instances on the same host are scrapped one by one, every
	// one of them has to be kept in the result
	fileStats := map[string][]CollectStat{}
	mergeFileStats(fileStats, map[string][]CollectStat{
		"127.0.0.1": {{Target: "/data/db/data/tikv-20160/rocksdb.info", Size: 1}},
	})
	mergeFileStats(fileStats, map[string][]CollectStat{
		"127.0.0.1": {{Target: "/data/db/data/tikv-20161/rocksdb.info", Size: 2}},
	})

	assert.Equal([]CollectStat{
		{Target: "/data/db/data/tikv-20160/rocksdb.info", Size: 1},
		{Target: "/data/db/data/tikv-20161/rocksdb.info", Size: 2},
	}, fileStats["127.0.0.1"])

	// hosts of other nodes are merged independently
	mergeFileStats(fileStats, map[string][]CollectStat{
		"127.0.0.2": {{Target: "/data/db/data/tikv-20160/rocksdb.info", Size: 3}},
	})
	assert.Len(fileStats["127.0.0.2"], 1)
	assert.Len(fileStats["127.0.0.1"], 2)
}
