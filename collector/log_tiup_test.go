package collector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/stretchr/testify/require"
)

// tiupTopology is the smallest topology that exercises the log collector: two
// TiKV instances on the same host, which is what makes the per instance rocksdb
// scraping - and therefore the trimming and the merging - worth checking.
const tiupTopology = `global:
  user: ubuntu
  ssh_port: 22
  deploy_dir: /data/db/deploy
  data_dir: /data/db/data
pd_servers:
  - host: 127.0.0.1
    client_port: 2379
tikv_servers:
  - host: 127.0.0.1
    port: 20160
    status_port: 20180
  - host: 127.0.0.1
    port: 20161
    status_port: 20181
`

func testTiUPTopology(t *testing.T) spec.Topology {
	t.Helper()
	return testTopology(t, tiupTopology)
}

// testTopology parses a topology the way a collection does.
func testTopology(t *testing.T, yaml string) spec.Topology {
	t.Helper()
	require.NoError(t, spec.Initialize("cluster"))

	f := filepath.Join(t.TempDir(), "topology.yaml")
	require.NoError(t, os.WriteFile(f, []byte(yaml), 0o644))

	topo := &spec.Specification{}
	require.NoError(t, spec.ParseTopologyYaml(f, topo))
	return topo
}

func testLogOptions(c collectLog) *LogCollectOptions {
	return &LogCollectOptions{
		BaseOptions: &BaseOptions{
			Cluster:     "test",
			User:        "ubuntu",
			ScrapeBegin: "2026-09-21T09:00:00+08:00",
			ScrapeEnd:   "2026-09-21T10:00:00+08:00",
		},
		opt:       &operator.Options{},
		collector: c,
		fileStats: map[string][]CollectStat{},
	}
}

func testManager() *Manager {
	return &Manager{
		mode:        CollectModeTiUP,
		specManager: spec.GetSpecManager(),
		logger:      logprinter.NewLogger(""),
	}
}

// renderSteps prints the steps the way the task framework does, which exposes
// the commands and the directories they work on.
func renderSteps(steps []*task.StepDisplay) string {
	var b strings.Builder
	for _, s := range steps {
		b.WriteString(s.String())
		b.WriteString("\n")
	}
	return b.String()
}

// TestBuildTiUPLogTasksWiring covers what the pure command builders can not:
// that the collector actually asks for those commands, once per TiKV instance,
// with the trimming on the rocksdb ones only. Deleting the trimming from the
// command of a step - or building the generic steps for a rocksdb only
// collection, which is what used to abort it - is visible here and nowhere else.
func TestBuildTiUPLogTasksWiring(t *testing.T) {
	topo := testTiUPTopology(t)
	m := testManager()

	t.Run("rocksdb only", func(t *testing.T) {
		assert := require.New(t)
		opt := testLogOptions(collectLog{Rocksdb: true})

		tasks, err := opt.buildTiUPLogTasks(m, topo)
		assert.NoError(err)

		rendered := renderSteps(tasks.scrape)
		assert.Equal(2, strings.Count(rendered, "--logtype rocksdb"),
			"one rocksdb scrape per TiKV instance on the host:\n%s", rendered)
		assert.Equal(2, strings.Count(rendered, "--trim --trim-dir '"+opt.trimDir()+"'"),
			"every rocksdb scrape trims into the directory the log collector owns:\n%s", rendered)
		assert.Contains(rendered, "--log '/data/db/data/tikv-20160/*'")
		assert.Contains(rendered, "--log '/data/db/data/tikv-20161/*'")

		// nothing scrapes the component log directories: an empty --logtype is
		// what made the scraper abort the whole collection
		assert.NotContains(rendered, "--logtype std")
		assert.NotContains(rendered, "--logtype slow")
		assert.NotContains(rendered, "--logtype unknown")
	})

	t.Run("unknown only", func(t *testing.T) {
		assert := require.New(t)
		opt := testLogOptions(collectLog{Unknown: true})

		tasks, err := opt.buildTiUPLogTasks(m, topo)
		assert.NoError(err)

		rendered := renderSteps(tasks.scrape)
		assert.Contains(rendered, "--logtype unknown")
		assert.NotContains(rendered, "--logtype rocksdb")
		assert.NotContains(rendered, "--trim", "component logs are collected whole on purpose")
	})
}

// TestBuildTiUPLogDownloadTasks checks the remote sources, package paths and
// successful-run tool cleanup. Private trim directories are instead released
// by Close, including when downloads fail; see log_cleanup_test.go.
func TestBuildTiUPLogDownloadTasks(t *testing.T) {
	assert := require.New(t)
	topo := testTiUPTopology(t)
	opt := testLogOptions(collectLog{Rocksdb: true})
	opt.resultDir = "/tmp/result"
	opt.fileStats = map[string][]CollectStat{
		"127.0.0.1": {
			// a trimmed copy: it has to land where the original file would be
			{Target: filepath.Join(opt.trimDir(), "data/db/data/tikv-20160/rocksdb.info"), Size: 10},
			// a file collected in place keeps its own path
			{Target: "/data/db/deploy/tikv-20160/log/tikv.log", Size: 20},
		},
	}

	download, clean, err := opt.buildTiUPLogDownloadTasks(testManager(), topo)
	assert.NoError(err)

	// the remote source stays under the trim directory, the local destination
	// is the original location inside the package
	rendered := renderSteps(download)
	assert.Contains(rendered,
		"remote=127.0.0.1:"+filepath.Join(opt.trimDir(), "data/db/data/tikv-20160/rocksdb.info"))
	assert.Contains(rendered,
		"local="+filepath.Join("/tmp/result", "127.0.0.1", "data/db/data/tikv-20160/rocksdb.info"))
	assert.Contains(rendered, "remote=127.0.0.1:/data/db/deploy/tikv-20160/log/tikv.log")
	assert.Contains(rendered,
		"local="+filepath.Join("/tmp/result", "127.0.0.1", "/data/db/deploy/tikv-20160/log/tikv.log"))

	cleaned := renderSteps(clean)
	assert.NotContains(cleaned, opt.trimDir(), "private directories are cleaned unconditionally by Close")
	assert.Contains(cleaned, task.CheckToolsPathDir)
}
