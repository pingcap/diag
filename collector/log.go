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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// componentDiagCollector is the component name of diagnostic collector
	componentDiagCollector = "diag"
)

type collectLog struct {
	Std     bool
	Slow    bool
	Unknown bool
	Ops     bool
}

// LogCollectOptions are options used collecting component logs
type LogCollectOptions struct {
	*BaseOptions
	collector collectLog
	opt       *operator.Options // global operations from cli
	limit     int               // scp rate limit
	resultDir string
	fileStats map[string][]CollectStat
	compress  bool
	kubeCli   *kubernetes.Clientset
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

// Prepare implements the Collector interface
func (c *LogCollectOptions) Prepare(m *Manager, cls *models.TiDBCluster) (map[string][]CollectStat, error) {
	switch m.mode {
	case CollectModeTiUP:
		if !(c.collector.Std || c.collector.Slow) {
			return nil, nil
		}
	case CollectModeK8s:
		return c.prepareK8s(m, cls)
	default:
		return nil, nil
	}

	topo := cls.Attributes[CollectModeTiUP].(spec.Topology)
	var (
		dryRunTasks   []*task.StepDisplay
		downloadTasks []*task.StepDisplay
	)
	diagcolVer := spec.TiDBComponentVersion(componentDiagCollector, "")

	uniqueHosts := map[string]int{}             // host -> ssh-port
	uniqueArchList := make(map[string]struct{}) // map["os-arch"]{}
	hostPaths := make(map[string]set.StringSet)
	hostTasks := make(map[string]*task.Builder)

	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	components := topo.ComponentsByUpdateOrder()
	components = operator.FilterComponent(components, roleFilter)

	for _, comp := range components {
		switch comp.Name() {
		case spec.ComponentGrafana,
			spec.ComponentAlertmanager,
			spec.ComponentTiSpark,
			spec.ComponentSpark:
			continue
		}
		instances := operator.FilterInstance(comp.Instances(), nodeFilter)
		if len(instances) < 1 {
			continue
		}

		for _, inst := range instances {
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

			// add filepaths to list
			if _, found := hostPaths[inst.GetHost()]; !found {
				hostPaths[inst.GetHost()] = set.NewStringSet()
			}
			hostPaths[inst.GetHost()].Insert(fmt.Sprintf("%s/*", inst.LogDir()))
		}
	}

	var scraperLogType []string
	if c.collector.Std {
		scraperLogType = append(scraperLogType, scraper.LogTypeStd)
	}
	if c.collector.Slow {
		scraperLogType = append(scraperLogType, scraper.LogTypeSlow)
	}
	if c.collector.Unknown {
		scraperLogType = append(scraperLogType, scraper.LogTypeUnknown)
	}

	// build scraper tasks
	for h, t := range hostTasks {
		host := h
		t = t.
			Shell(
				host,
				fmt.Sprintf("%s --log '%s' -f '%s' -t '%s' --logtype %s",
					filepath.Join(task.CheckToolsPathDir, "bin", "scraper"),
					strings.Join(hostPaths[host].Slice(), ","),
					c.ScrapeBegin, c.ScrapeEnd,
					strings.Join(scraperLogType, ","),
				),
				"",
				false,
			).
			Func(
				host,
				func(ctx context.Context) error {
					stats, err := parseScraperSamples(ctx, host)
					if err != nil {
						return err
					}
					for host, files := range stats {
						c.fileStats[host] = files
					}
					return nil
				},
			)
		t1 := t.BuildAsStep(fmt.Sprintf("  - Scraping log files on %s:%d", host, uniqueHosts[host]))
		dryRunTasks = append(dryRunTasks, t1)
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Download necessary tools", false, downloadTasks...).
		ParallelStep("+ Collect host information", false, dryRunTasks...).
		Build()

	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)
	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return nil, err
		}
		return nil, perrs.Trace(err)
	}

	return c.fileStats, nil
}

// Collect implements the Collector interface
func (c *LogCollectOptions) Collect(m *Manager, cls *models.TiDBCluster) error {
	switch m.mode {
	case CollectModeTiUP:
	case CollectModeK8s:
		return c.collectK8s(m, cls)
	default:
		return nil
	}

	topo := cls.Attributes[CollectModeTiUP].(spec.Topology)
	var (
		collectTasks []*task.StepDisplay
		cleanTasks   []*task.StepDisplay
	)
	uniqueHosts := map[string]int{} // host -> ssh-port

	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	components := topo.ComponentsByUpdateOrder()
	components = operator.FilterComponent(components, roleFilter)

	for _, comp := range components {
		switch comp.Name() {
		case spec.ComponentGrafana,
			spec.ComponentAlertmanager,
			spec.ComponentTiSpark,
			spec.ComponentSpark:
			continue
		}
		instances := operator.FilterInstance(comp.Instances(), nodeFilter)
		if len(instances) < 1 {
			continue
		}

		for _, inst := range instances {
			// checks that applies to each host
			if _, found := uniqueHosts[inst.GetHost()]; found {
				continue
			}
			uniqueHosts[inst.GetHost()] = inst.GetSSHPort()

			t2, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
			if err != nil {
				return err
			}
			for _, f := range c.fileStats[inst.GetHost()] {
				// build checking tasks
				t2 = t2.
					// check for listening ports
					CopyFile(
						f.Target,
						filepath.Join(c.resultDir, inst.GetHost(), f.Target),
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
				return err
			}
			t3 := b.
				Rmdir(inst.GetHost(), task.CheckToolsPathDir).
				BuildAsStep(fmt.Sprintf("  - Cleanup temp files on %s:%d", inst.GetHost(), inst.GetSSHPort()))
			cleanTasks = append(cleanTasks, t3)
		}
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
	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
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

		var logs []CollectStat
		if c.collector.Std {
			logs = append(logs, CollectStat{
				Target: podName + ".log",
				Attributes: map[string]interface{}{
					"podName":       podName,
					"containerName": string(inst.Type()),
				},
			})
		}
		if c.collector.Slow && inst.Type() == models.ComponentTypeTiDB {
			logs = append(logs, CollectStat{
				Target: "tidb_slow.log",
				Attributes: map[string]interface{}{
					"podName":       podName,
					"containerName": "slowlog",
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

			req := c.kubeCli.CoreV1().Pods(c.Namespace).GetLogs(podName, &opt)

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

			r := bufio.NewReader(stream)
			w := bufio.NewWriter(f)
			for {
				bytes, err := r.ReadBytes('\n')
				if err == io.EOF {
					break
				} else if err != nil {
					return err
				}

				_, err = w.Write(bytes)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil

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
