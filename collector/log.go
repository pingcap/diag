// Copyright 2021 PingCAP, Inc.
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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/joomcode/errorx"
	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/diag/scraper"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/tui"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// componentDiagCollector is the component name of diagnostic collector
	componentDiagCollector = "diag"

	// trimDirPrefix is the first part of the name of every directory holding
	// trimmed logs. Together with trimDirRoot it is the whole rule deciding
	// what this collector is allowed to remove on a target host, so leftovers
	// of an interrupted collection are matched by name and never by position.
	trimDirPrefix = "diag-trimmed-"
)

// trimDirRoot is the directory holding the trimmed logs: private to this
// collection and outside the shared tools directory, which other collectors can
// remove before the log download starts.
func trimDirRoot() string {
	return filepath.Dir(task.CheckToolsPathDir)
}

// trimDir is the directory this run trims into. The random suffix keeps the
// copies of one run apart from the leftovers of another one, so a run never
// removes what it did not create unless the user asks for it.
func (c *LogCollectOptions) trimDir() string {
	if c.trimPath == "" {
		c.trimPath = filepath.Join(trimDirRoot(), trimDirPrefix+uuid.NewString())
	}
	return c.trimPath
}

// isTrimDirName reports whether name is a name trimDir() can produce. The
// suffix is the uuid of the run that created the directory, which keeps a file
// that merely starts with the prefix from being taken for a trimmed logs
// directory.
func isTrimDirName(name string) bool {
	suffix, found := strings.CutPrefix(name, trimDirPrefix)
	if !found {
		return false
	}
	_, err := uuid.Parse(suffix)
	return err == nil
}

// assertRemovableTrimDir refuses to remove anything but a directory the
// trimming rule created: one level below trimDirRoot, named trimDirPrefix
// followed by a uuid. Removing a path that does not match the rule means the
// rule changed, and deleting it could destroy data this collector does not own.
func assertRemovableTrimDir(dir string) error {
	switch {
	case dir == "" || !filepath.IsAbs(dir) || filepath.Clean(dir) != dir:
		return fmt.Errorf("refusing to remove %q: not a clean absolute path", dir)
	case filepath.Dir(dir) != trimDirRoot() || !isTrimDirName(filepath.Base(dir)):
		return fmt.Errorf("refusing to remove %q: only %s*%s directories in %s are created by log trimming",
			dir, trimDirPrefix, "<uuid>", trimDirRoot())
	}
	return nil
}

// prepareTrimDir registers cleanup only after this run creates the directory.
// Keep the executor from Prepare so Close also works if Collect is never called.
func (c *LogCollectOptions) prepareTrimDir(host string) *task.Func {
	dir := c.trimDir()
	return task.NewFunc("prepare private trim directory", func(ctx context.Context) error {
		exec, ok := ctxt.GetInner(ctx).GetExecutor(host)
		if !ok {
			return task.ErrNoExecutor
		}
		if _, _, err := exec.Execute(ctx, "mkdir -m 700 -- '"+dir+"'", false); err != nil {
			return err
		}
		c.trimMu.Lock()
		defer c.trimMu.Unlock()
		if c.trimCleanups == nil {
			c.trimCleanups = make(map[string]func())
		}
		c.trimCleanups[host] = func() {
			if _, _, err := exec.Execute(context.Background(), "rm -rf -- '"+dir+"'", false); err != nil {
				ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger).Warnf("Failed to clean trimmed logs on %s in %s: %v", host, dir, err)
			}
		}
		return nil
	})
}

// Close releases only the directories created by this collection. It is safe
// to call after Collect and again when the manager returns (including aborts).
func (c *LogCollectOptions) Close() {
	c.trimMu.Lock()
	cleanups := c.trimCleanups
	c.trimCleanups = nil
	c.trimMu.Unlock()
	for _, cleanup := range cleanups {
		cleanup()
	}
}

// pathInPackage returns the path a collected file gets inside the package. A
// trimmed copy is reported by the scraper with its absolute temporary path,
// which must not leak into the package: it is placed where the original file
// would have been.
func (c *LogCollectOptions) pathInPackage(resultDir, host, target string) string {
	rel, err := filepath.Rel(c.trimDir(), target)
	if err == nil && rel != "." && rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.Join(resultDir, host, rel)
	}
	return filepath.Join(resultDir, host, target)
}

type collectLog struct {
	Std     bool
	Slow    bool
	Unknown bool
	Ops     bool
	Rocksdb bool
}

// LogCollectOptions are options used collecting component logs
type LogCollectOptions struct {
	*BaseOptions
	collector     collectLog
	opt           *operator.Options // global operations from cli
	limit         int               // scp rate limit
	resultDir     string
	fileStats     map[string][]CollectStat
	compress      bool
	kubeCli       *kubernetes.Clientset
	skipConfirm   bool // assume yes to every confirmation, as -y does
	cleanLeftover bool // remove leftover trimmed logs without asking
	trimPath      string
	trimMu        sync.Mutex
	trimCleanups  map[string]func()
}

// Desc implements the Collector interface
func (c *LogCollectOptions) Desc() string {
	return "logs of components"
}

// GetBaseOptions implements the Collector interface
func (c *LogCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *LogCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *LogCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *LogCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// logTypesToScrap returns the types of logs to be scrapped from the component
// log directories. RocksDB logs are not part of it: they live in the TiKV data
// directory and are collected by a dedicated step, so the generic scraper step
// must not be built at all when this returns an empty list (scraper rejects a
// flag without value: "flag needs an argument: --logtype").
func (c *LogCollectOptions) logTypesToScrap() []string {
	var logTypes []string
	if c.collector.Std {
		logTypes = append(logTypes, scraper.LogTypeStd)
	}
	if c.collector.Slow {
		logTypes = append(logTypes, scraper.LogTypeSlow)
	}
	if c.collector.Unknown {
		logTypes = append(logTypes, scraper.LogTypeUnknown)
	}
	return logTypes
}

// needsCollect reports whether the collector has anything to do. It derives
// from logTypesToScrap so that the two can not drift apart: keeping them in
// sync by hand is how log.unknown ended up being collected by nobody.
func (c *LogCollectOptions) needsCollect() bool {
	return len(c.logTypesToScrap()) > 0 || c.collector.Rocksdb
}

// leftoverTrimmedLog is one directory of trimmed logs an earlier collection
// left on a host, with the disk it still occupies.
type leftoverTrimmedLog struct {
	path string
	size int64
}

// leftoverTrimmedLogsCmdIn prints one "<path>\t<size in KiB>" line per
// directory that could hold trimmed logs below root. It matches by name - the
// name trimming gives its directories - and touches nothing: whether a matched
// entry really belongs to this collector is decided by assertRemovableTrimDir,
// in one place, before anything is removed. du -sk is POSIX, -b would need GNU.
//
// The loop is wrapped in sh -c to make the whole thing a single simple command.
// Every executor prepends an assignment without a separator before the command
// it runs (PATH=$PATH:/bin:... <command>), and a compound command can not follow
// one: the host would report "syntax error near unexpected token `do'".
func leftoverTrimmedLogsCmdIn(root string) string {
	script := fmt.Sprintf(
		`for d in "%s"/%s*; do if [ -d "$d" ]; then printf "%%s\t%%s\n" "$d" "$(du -sk "$d" 2>/dev/null | cut -f1)"; fi; done`,
		root, trimDirPrefix)
	return "sh -c '" + script + "'"
}

// leftoverTrimmedLogsCmd looks for trimmed logs where this run would put them.
func leftoverTrimmedLogsCmd() string {
	return leftoverTrimmedLogsCmdIn(trimDirRoot())
}

// parseLeftoverTrimmedLogs reads the output of leftoverTrimmedLogsCmd and turns
// the reported kibibytes into bytes. A line it can not make sense of is ignored,
// and an unreadable size counts as 0: a host that fails to report leftovers then
// looks like a host without any, which only means the user is not offered to
// remove them.
func parseLeftoverTrimmedLogs(out string) []leftoverTrimmedLog {
	var logs []leftoverTrimmedLog
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r\n")
		path, size, found := strings.Cut(line, "\t")
		if !found {
			continue
		}
		if path = strings.TrimSpace(path); path == "" {
			continue
		}
		kib, err := strconv.ParseInt(strings.TrimSpace(size), 10, 64)
		if err != nil || kib < 0 {
			kib = 0
		}
		logs = append(logs, leftoverTrimmedLog{path: path, size: kib * 1024})
	}
	return logs
}

// probeLeftoverTrimmedLogs looks for the trimmed logs an interrupted collection
// left on hosts, so that the user can be asked about them before this run adds
// its own.
func (c *LogCollectOptions) probeLeftoverTrimmedLogs(ctx context.Context, m *Manager, topo spec.Topology, hosts []string) (map[string][]leftoverTrimmedLog, error) {
	steps := make([]*task.StepDisplay, 0, len(hosts))
	for _, host := range hosts {
		b, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
		if err != nil {
			return nil, err
		}
		steps = append(steps, b.
			Shell(host, leftoverTrimmedLogsCmd(), "", false).
			BuildAsStep(fmt.Sprintf("  - Looking for leftover trimmed logs on %s", host)))
	}
	if len(steps) == 0 {
		return nil, nil
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Looking for leftover trimmed logs", false, steps...).
		Build()
	if err := t.Execute(ctx); err != nil {
		return nil, perrs.Trace(err)
	}

	leftovers := make(map[string][]leftoverTrimmedLog)
	for _, host := range hosts {
		stdout, _, _ := ctxt.GetInner(ctx).GetOutputs(host)
		if logs := parseLeftoverTrimmedLogs(string(stdout)); len(logs) > 0 {
			leftovers[host] = logs
		}
	}
	return leftovers, nil
}

// ownedTrimmedLogs keeps the entries the removal rule accepts - the directories
// this version of the trimming created - and returns the others apart, so an
// entry that only looks like one is reported instead of being removed.
func ownedTrimmedLogs(logs []leftoverTrimmedLog) (owned, foreign []leftoverTrimmedLog) {
	for _, l := range logs {
		if err := assertRemovableTrimDir(l.path); err != nil {
			foreign = append(foreign, l)
			continue
		}
		owned = append(owned, l)
	}
	return owned, foreign
}

// describeLeftoverTrimmedLogs sums up what was found, one line per host.
func describeLeftoverTrimmedLogs(hosts []string, leftovers map[string][]leftoverTrimmedLog) (string, int64) {
	var total int64
	var b strings.Builder
	for _, host := range hosts {
		logs, found := leftovers[host]
		if !found {
			continue
		}
		var size int64
		parts := make([]string, 0, len(logs))
		for _, l := range logs {
			size += l.size
			parts = append(parts, fmt.Sprintf("%s (%s)", l.path, readableSize(l.size)))
		}
		total += size
		fmt.Fprintf(&b, "  %s: %s\n", host, strings.Join(parts, ", "))
	}
	return b.String(), total
}

// confirmLeftoverTrimmedRemoval asks whether the leftovers may be removed. -y
// answers yes to every confirmation, so it does not ask; neither does the flag
// that asks for the removal explicitly. What was found is reported by the
// caller, so it is printed once, whether or not there is anyone to ask.
func (c *LogCollectOptions) confirmLeftoverTrimmedRemoval() bool {
	if c.cleanLeftover || c.skipConfirm {
		return true
	}
	ok, _ := confirmRemoveLeftover("Remove these candidate trimmed logs before collecting?")
	return ok
}

// confirmRemoveLeftover is the prompt used to decide about the leftovers, kept
// in a variable so that both answers can be exercised.
var confirmRemoveLeftover = tui.PromptForConfirmYes

// buildLeftoverRemovalSteps builds one step per host holding leftovers, removing
// exactly the directories given and nothing else: every path is checked against
// the trimming rule first, and a path that fails the check aborts the whole
// removal rather than removing something this collector does not own.
func (c *LogCollectOptions) buildLeftoverRemovalSteps(m *Manager, topo spec.Topology, hosts []string, leftovers map[string][]leftoverTrimmedLog) ([]*task.StepDisplay, error) {
	steps := make([]*task.StepDisplay, 0, len(hosts))
	for _, host := range hosts {
		logs, found := leftovers[host]
		if !found {
			continue
		}
		for _, l := range logs {
			if err := assertRemovableTrimDir(l.path); err != nil {
				return nil, err
			}
		}

		b, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
		if err != nil {
			return nil, err
		}
		for _, l := range logs {
			m.logger.Infof("Removing leftover trimmed logs on %s: %s (%s)", host, l.path, readableSize(l.size))
			b = b.Rmdir(host, l.path)
		}
		steps = append(steps, b.BuildAsStep(fmt.Sprintf("  - Removing leftover trimmed logs on %s", host)))
	}
	return steps, nil
}

// removeLeftoverTrimmedLogs removes the leftover directories of hosts.
func (c *LogCollectOptions) removeLeftoverTrimmedLogs(ctx context.Context, m *Manager, topo spec.Topology, hosts []string, leftovers map[string][]leftoverTrimmedLog) error {
	steps, err := c.buildLeftoverRemovalSteps(m, topo, hosts, leftovers)
	if err != nil {
		return err
	}
	if len(steps) == 0 {
		return nil
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Removing leftover trimmed logs", false, steps...).
		Build()
	if err := t.Execute(ctx); err != nil {
		return perrs.Trace(err)
	}
	return nil
}

// cleanLeftoverTrimmedLogs removes trimmed logs that may have been left by an
// interrupted collection. A UUID identifies a directory but does not prove its
// collection has stopped: a concurrent collection may still be using it.
// TODO: Coordinate active collections (for example, with a lease) before
// reclaiming directories, so an active collection cannot lose its files.
// For now, reclaiming leftovers takes priority. A warning is always emitted;
// -y and --clean-leftover-trimmed-logs skip the confirmation prompt.
func (c *LogCollectOptions) cleanLeftoverTrimmedLogs(ctx context.Context, m *Manager, topo spec.Topology) error {
	hosts := c.logHosts(topo)
	if len(hosts) == 0 {
		return nil
	}

	leftovers, err := c.probeLeftoverTrimmedLogs(ctx, m, topo, hosts)
	if err != nil {
		return err
	}

	// Only the directories the trimming rule names are this collector's to
	// remove. Anything else the probe matched - a directory of an older or a
	// newer naming, a file starting like one - is left alone and reported, so
	// that a change of the rule can not turn into removing foreign data.
	owned := make(map[string][]leftoverTrimmedLog)
	for host, logs := range leftovers {
		ours, foreign := ownedTrimmedLogs(logs)
		if len(foreign) > 0 {
			paths := make([]string, 0, len(foreign))
			for _, l := range foreign {
				paths = append(paths, l.path)
			}
			m.logger.Warnf("Not touching %s on %s: log trimming did not create it, remove it by hand if it is not needed",
				strings.Join(paths, ", "), host)
		}
		if len(ours) > 0 {
			owned[host] = ours
		}
	}
	if len(owned) == 0 {
		return nil
	}

	desc, total := describeLeftoverTrimmedLogs(hosts, owned)
	m.logger.Warnf("Found %s of trimmed logs in %s (some may belong to an active concurrent collection):\n%s",
		readableSize(total), trimDirRoot(), strings.TrimRight(desc, "\n"))
	m.logger.Warnf("Removing these directories may interrupt a concurrent RocksDB log collection")

	if !c.confirmLeftoverTrimmedRemoval() {
		m.logger.Warnf("Keeping them: they are not part of this collection, and a later one will ask again")
		return nil
	}
	if err := c.removeLeftoverTrimmedLogs(ctx, m, topo, hosts, owned); err != nil {
		return err
	}
	m.logger.Infof("Removed candidate trimmed logs")
	return nil
}

// scraperPath is where the scraper binary is found on the target hosts.
func scraperPath() string {
	return filepath.Join(task.CheckToolsPathDir, "bin", "scraper")
}

// consideredInstances returns the instances this collector works on, in start
// order: the components carrying logs, filtered by the --role and --node
// options. Both building the collection steps and looking for leftovers of an
// earlier collection go through it, so they can not disagree on which hosts
// hold data of this collector.
func (c *LogCollectOptions) consideredInstances(topo spec.Topology) []spec.Instance {
	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	components := operator.FilterComponent(topo.ComponentsByStartOrder(), roleFilter)

	var instances []spec.Instance
	for _, comp := range components {
		switch comp.Name() {
		case spec.ComponentGrafana,
			spec.ComponentAlertmanager,
			spec.ComponentTiSpark,
			spec.ComponentSpark:
			continue
		}
		instances = append(instances, operator.FilterInstance(comp.Instances(), nodeFilter)...)
	}
	return instances
}

// logHosts returns the hosts consideredInstances live on, without repetition,
// in the order they are met.
func (c *LogCollectOptions) logHosts(topo spec.Topology) []string {
	var hosts []string
	seen := map[string]struct{}{}
	for _, inst := range c.consideredInstances(topo) {
		if _, found := seen[inst.GetHost()]; found {
			continue
		}
		seen[inst.GetHost()] = struct{}{}
		hosts = append(hosts, inst.GetHost())
	}
	return hosts
}

// genericScraperCmd builds the command scraping the component log directories.
// ok is false when no type of those directories is requested: the step then
// must not be built at all, because the scraper is invoked with a bare
// --logtype and pflag rejects a flag without value with
// "flag needs an argument: --logtype", which aborts the whole collection.
func genericScraperCmd(paths []string, begin, end string, logTypes []string) (cmd string, ok bool) {
	if len(logTypes) == 0 {
		return "", false
	}
	return fmt.Sprintf("%s --log '%s' -f '%s' -t '%s' --logtype %s",
		scraperPath(), strings.Join(paths, ","), begin, end,
		strings.Join(logTypes, ",")), true
}

// rocksdbScraperCmd builds the command scraping the rocksdb logs of one TiKV
// data directory. --trim keeps only the lines inside [begin, end]: a data
// directory accumulates one rocksdb.info per rotation and TiKV never deletes
// the rotated ones by default (log.file.max-backups and max-days are 0), so
// shipping them in full is both slow and useless.
func rocksdbScraperCmd(dataDir, begin, end, outDir string) string {
	return fmt.Sprintf("%s --log '%s/*' -f '%s' -t '%s' --logtype %s --trim --trim-dir '%s'",
		scraperPath(), dataDir, begin, end, scraper.LogTypeRocksDB, outDir)
}

// collectScrapedStats reads the sample the scraper printed for host and merges
// it into fileStats. Every scraper step ends here, so a host that runs the
// scraper several times - once per TiKV instance for the rocksdb logs, plus
// once for the component log directories - accumulates all the results instead
// of keeping only the last one.
func (c *LogCollectOptions) collectScrapedStats(ctx context.Context, host string) error {
	stats, err := parseScraperSamples(ctx, host)
	if err != nil {
		return err
	}
	mergeFileStats(c.fileStats, stats)
	return nil
}

// Prepare implements the Collector interface
func (c *LogCollectOptions) Prepare(m *Manager, cls *models.TiDBCluster) (map[string][]CollectStat, error) {
	switch m.mode {
	case CollectModeTiUP:
		if !c.needsCollect() {
			return nil, nil
		}
	case CollectModeK8s:
		return c.prepareK8s(m, cls)
	default:
		return nil, nil
	}

	topo := cls.Attributes[CollectModeTiUP].(spec.Topology)
	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)

	// Trimming leaves its copies behind if a collection dies before it
	// downloads them, and only trimming knows where they are: ask about them
	// before this run adds more.
	if c.collector.Rocksdb {
		if err := c.cleanLeftoverTrimmedLogs(ctx, m, topo); err != nil {
			// the manager closes every collector even when Prepare fails
			return nil, perrs.Trace(err)
		}
	}

	tasks, err := c.buildTiUPLogTasks(m, topo)
	if err != nil {
		return nil, err
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Download necessary tools", false, tasks.download...).
		ParallelStep("+ Collect host information", false, tasks.scrape...).
		Build()

	if err := t.Execute(ctx); err != nil {
		c.Close()
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return nil, err
		}
		return nil, perrs.Trace(err)
	}

	return c.fileStats, nil
}

// tiUPLogTasks are the steps of one log collection on a tiup deployed cluster:
// the ones that put the collecting tools on the hosts, and the ones that scrape
// the logs. They are built apart from Prepare so that the wiring - which
// commands are generated for which host, and what is cleaned up afterwards -
// can be checked without a cluster.
type tiUPLogTasks struct {
	download []*task.StepDisplay
	scrape   []*task.StepDisplay
}

// buildTiUPLogTasks builds the steps of a tiup collection without running them.
func (c *LogCollectOptions) buildTiUPLogTasks(m *Manager, topo spec.Topology) (*tiUPLogTasks, error) {
	var (
		dryRunTasks   []*task.StepDisplay
		downloadTasks []*task.StepDisplay
	)
	diagcolVer := spec.TiDBComponentVersion(componentDiagCollector, "")

	uniqueHosts := map[string]int{}             // host -> ssh-port
	uniqueArchList := make(map[string]struct{}) // map["os-arch"]{}
	hostPaths := make(map[string]set.StringSet)
	hostTasks := make(map[string]*task.Builder)
	trimHosts := make(map[string]bool)

	for _, inst := range c.consideredInstances(topo) {
		archKey := fmt.Sprintf("%s-%s", inst.OS(), inst.Arch())
		if _, found := uniqueArchList[archKey]; !found {
			uniqueArchList[archKey] = struct{}{}
			t0 := task.NewBuilder(m.logger).
				Download(
					componentDiagCollector,
					inst.OS(),
					inst.Arch(),
					diagcolVer,
				).
				BuildAsStep(fmt.Sprintf("  - Downloading collecting tools for %s/%s", inst.OS(), inst.Arch()))
			downloadTasks = append(downloadTasks, t0)
		}

		// tasks that applies to each host
		if _, found := uniqueHosts[inst.GetHost()]; !found {
			uniqueHosts[inst.GetHost()] = inst.GetSSHPort()
			// build system info collecting tasks
			t1, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
			if err != nil {
				return nil, err
			}
			t1 = t1.
				Mkdir(c.GetBaseOptions().User, inst.GetHost(), filepath.Join(task.CheckToolsPathDir, "bin")).
				CopyComponent(
					componentDiagCollector,
					inst.OS(),
					inst.Arch(),
					diagcolVer,
					"", // use default srcPath
					inst.GetHost(),
					task.CheckToolsPathDir,
				)
			hostTasks[inst.GetHost()] = t1
		}

		// Placing this code here is not elegant, but it can avoid encountering unknown files from collecting datadir.
		if c.collector.Rocksdb && inst.ComponentName() == spec.ComponentTiKV {
			// --trim keeps only the rocksdb lines inside the requested time
			// range. A data directory accumulates one rocksdb.info per
			// rotation and TiKV never deletes the rotated ones
			// (log.file.max-backups defaults to 0), so collecting them in
			// full is both slow and useless.
			host := inst.GetHost()
			if !trimHosts[host] {
				trimHosts[host] = true
				hostTasks[host].Func(host, c.prepareTrimDir(host).Execute)
			}
			hostTasks[host].
				Shell(
					host,
					rocksdbScraperCmd(inst.DataDir(), c.ScrapeBegin, c.ScrapeEnd, c.trimDir()),
					"",
					false,
				).
				Func(
					host,
					func(ctx context.Context) error {
						return c.collectScrapedStats(ctx, host)
					},
				)
		}

		// add filepaths to list
		if _, found := hostPaths[inst.GetHost()]; !found {
			hostPaths[inst.GetHost()] = set.NewStringSet()
		}
		hostPaths[inst.GetHost()].Insert(fmt.Sprintf("%s/*", inst.LogDir()))
	}

	scraperLogType := c.logTypesToScrap()

	// build scraper tasks
	for h, t := range hostTasks {
		host := h
		// Only build the generic scraper step when at least one log type of the
		// component log directories (std/slow/unknown) is requested. Otherwise
		// the scraper would be invoked with an empty --logtype value, which
		// makes it fail with "flag needs an argument: --logtype" and aborts the
		// whole collection (e.g. `--include=log.rocksdb`).
		if cmd, ok := genericScraperCmd(hostPaths[host].Slice(), c.ScrapeBegin, c.ScrapeEnd, scraperLogType); ok {
			t = t.
				Shell(
					host,
					cmd,
					"",
					false,
				).
				Func(
					host,
					func(ctx context.Context) error {
						return c.collectScrapedStats(ctx, host)
					},
				)
		}
		t1 := t.BuildAsStep(fmt.Sprintf("  - Scraping log files on %s:%d", host, uniqueHosts[host]))
		dryRunTasks = append(dryRunTasks, t1)
	}

	return &tiUPLogTasks{download: downloadTasks, scrape: dryRunTasks}, nil
}

// Collect implements the Collector interface
func (c *LogCollectOptions) Collect(m *Manager, cls *models.TiDBCluster) error {
	defer c.Close()
	switch m.mode {
	case CollectModeTiUP:
	case CollectModeK8s:
		return c.collectK8s(m, cls)
	default:
		return nil
	}

	topo := cls.Attributes[CollectModeTiUP].(spec.Topology)
	collectTasks, cleanTasks, err := c.buildTiUPLogDownloadTasks(m, topo)
	if err != nil {
		return err
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Scrap files on nodes", false, collectTasks...).
		ParallelStep("+ Cleanup temp files", false, cleanTasks...).
		Build()

	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)
	if err := c.runLogDownload(ctx, t); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

// runLogDownload keeps cleanup independent of the serial task's success path.
func (c *LogCollectOptions) runLogDownload(ctx context.Context, t task.Task) error {
	defer c.Close()
	return t.Execute(ctx)
}

// buildTiUPLogDownloadTasks builds the steps that download the files found on
// the hosts and the ones that remove the temporary directories afterwards,
// without running them.
func (c *LogCollectOptions) buildTiUPLogDownloadTasks(m *Manager, topo spec.Topology) (collectTasks, cleanTasks []*task.StepDisplay, err error) {
	uniqueHosts := map[string]int{} // host -> ssh-port

	for _, inst := range c.consideredInstances(topo) {
		// checks that applies to each host
		if _, found := uniqueHosts[inst.GetHost()]; found {
			continue
		}
		uniqueHosts[inst.GetHost()] = inst.GetSSHPort()

		t2, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
		if err != nil {
			return nil, nil, err
		}
		for _, f := range c.fileStats[inst.GetHost()] {
			// build checking tasks
			t2 = t2.
				// check for listening ports
				CopyFile(
					f.Target,
					c.pathInPackage(c.resultDir, inst.GetHost(), f.Target),
					inst.GetHost(),
					true,
					c.limit,
					c.compress,
				)
		}
		collectTasks = append(
			collectTasks,
			t2.BuildAsStep(fmt.Sprintf("  - Downloading log files from node %s", inst.GetHost())),
		)

		b, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
		if err != nil {
			return nil, nil, err
		}
		// Trimmed copies are removed by Close, even if downloading fails.
		// Preserve the existing successful-run cleanup of collecting tools.
		t3 := b.
			Rmdir(inst.GetHost(), task.CheckToolsPathDir).
			BuildAsStep(fmt.Sprintf("  - Cleanup temp files on %s:%d", inst.GetHost(), inst.GetSSHPort()))
		cleanTasks = append(cleanTasks, t3)
	}

	return collectTasks, cleanTasks, nil
}

func (c *LogCollectOptions) prepareK8s(m *Manager, cls *models.TiDBCluster) (map[string][]CollectStat, error) {
	roleFilter := set.NewStringSet(c.opt.Roles...)
	comps := cls.Components()
	comps = models.FilterComponent(comps, roleFilter)

	c.fileStats = make(map[string][]CollectStat)

	for _, inst := range comps {
		podName, ok := inst.Attributes()["pod"].(string)
		if !ok {
			// return fmt.Errorf("pod name not found in %s", inst.ID())
			// component like prometheus does not have pod name
			continue
		}
		ns := inst.Attributes()["namespace"].(string)

		var logs []CollectStat
		if c.collector.Std {
			logs = append(logs, CollectStat{
				Target: string(inst.Type()) + ".log",
				Attributes: map[string]interface{}{
					"podName":       podName,
					"containerName": string(inst.Type()),
					"namespace":     ns,
				},
			})
		}
		if c.collector.Slow && inst.Type() == models.ComponentTypeTiDB {
			logs = append(logs, CollectStat{
				Target: "tidb_slow_query.log",
				Attributes: map[string]interface{}{
					"podName":       podName,
					"containerName": "slowlog",
					"namespace":     ns,
				},
			})
		}
		c.fileStats[podName] = logs
	}
	return c.fileStats, nil
}

func (c *LogCollectOptions) collectK8s(m *Manager, cls *models.TiDBCluster) error {
	beginTime, _ := utils.ParseTime(c.GetBaseOptions().ScrapeBegin)

	for podName, fileStats := range c.fileStats {
		for _, fs := range fileStats {
			opt := corev1.PodLogOptions{
				Container: fs.Attributes["containerName"].(string),
				SinceTime: &metav1.Time{Time: beginTime},
			}

			req := c.kubeCli.CoreV1().Pods(fs.Attributes["namespace"].(string)).GetLogs(podName, &opt)

			stream, err := req.Stream(context.TODO())
			if err != nil {
				return err
			}
			defer stream.Close()

			fp := filepath.Join(c.resultDir, "logs", podName, fs.Target)
			err = os.MkdirAll(filepath.Dir(fp), 0755)
			if err != nil {
				return err
			}
			f, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, stream)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// mergeFileStats merges the result of one scraper run into fileStats. A host
// may run the scraper several times (once per TiKV instance for rocksdb logs,
// and once for the component log directories), so results must be accumulated
// instead of overwritten, otherwise only the last run would be collected.
func mergeFileStats(fileStats map[string][]CollectStat, stats map[string][]CollectStat) {
	for host, files := range stats {
		if fileStats[host] == nil {
			fileStats[host] = files
		} else {
			fileStats[host] = append(fileStats[host], files...)
		}
	}
}

func parseScraperSamples(ctx context.Context, host string) (map[string][]CollectStat, error) {
	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)
	stdout, stderr, _ := ctxt.GetInner(ctx).GetOutputs(host)
	if len(stderr) > 0 {
		logger.Errorf("error scraping files: %s, logs might be incomplete", stderr)
	}
	if len(stdout) < 1 {
		// no matched files, just skip
		return nil, nil
	}

	var s scraper.Sample
	if err := json.Unmarshal(stdout, &s); err != nil {
		// save output directly on parsing errors
		return nil, fmt.Errorf("error parsing scraped stats: %s", stdout)
	}

	stats := make(map[string][]CollectStat)
	if _, found := stats[host]; !found {
		stats[host] = make([]CollectStat, 0)
	}

	for k, v := range s.Config {
		stats[host] = append(stats[host], CollectStat{
			Target: k,
			Size:   v,
		})
	}
	for k, v := range s.Log {
		stats[host] = append(stats[host], CollectStat{
			Target: k,
			Size:   v,
		})
	}
	for k, v := range s.TSDB {
		stats[host] = append(stats[host], CollectStat{
			Target: k,
			Size:   v,
		})
	}

	return stats, nil
}
