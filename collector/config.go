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
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joomcode/errorx"
	"github.com/pingcap/diag/pkg/models"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/utils"
)

// ConfigCollectOptions are options used collecting component logs
type ConfigCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	limit     int               // scp rate limit
	resultDir string
	fileStats map[string][]CollectStat
}

// Desc implements the Collector interface
func (c *ConfigCollectOptions) Desc() string {
	return "config files of components"
}

// GetBaseOptions implements the Collector interface
func (c *ConfigCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *ConfigCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *ConfigCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *ConfigCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *ConfigCollectOptions) Prepare(m *Manager, topo *models.TiDBCluster) (map[string][]CollectStat, error) {
	switch m.mode {
	case CollectModeTiUP:
		return c.prepareForTiUP(m, topo)
	}

	return nil, nil
}

// prepareForTiUP implements preparation for tiup-cluster deployed clusters
func (c *ConfigCollectOptions) prepareForTiUP(m *Manager, topo *models.TiDBCluster) (map[string][]CollectStat, error) {
	rawTopo := topo.Attributes[CollectModeTiUP].(*spec.Specification)

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
	components := topo.Components()
	components = models.FilterComponent(components, roleFilter)
	instances := models.FilterInstance(components, nodeFilter)

	for _, inst := range instances {
		switch inst.Type() {
		case models.ComponentTypeMonitor,
			models.ComponentTypeTiSpark:
			continue
		}

		os := inst.Attributes()["os"].(string)
		arch := inst.Attributes()["arch"].(string)
		archKey := fmt.Sprintf("%s-%s", os, arch)
		if _, found := uniqueArchList[archKey]; !found {
			uniqueArchList[archKey] = struct{}{}
			t0 := task.NewBuilder().
				Download(
					componentDiagCollector,
					os,
					arch,
					diagcolVer,
				).
				BuildAsStep(fmt.Sprintf("  - Downloading collecting tools for %s/%s", os, arch))
			downloadTasks = append(downloadTasks, t0)
		}

		// tasks that applies to each host
		if _, found := uniqueHosts[inst.Host()]; !found {
			uniqueHosts[inst.Host()] = inst.SSHPort()
			// build system info collecting tasks
			b, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, rawTopo, c.GetBaseOptions().User, *c.opt)
			if err != nil {
				return nil, err
			}
			t1 := b.
				Mkdir(c.GetBaseOptions().User, inst.Host(), filepath.Join(task.CheckToolsPathDir, "bin")).
				CopyComponent(
					componentDiagCollector,
					os,
					arch,
					diagcolVer,
					"", // use default srcPath
					inst.Host(),
					task.CheckToolsPathDir,
				)
			hostTasks[inst.Host()] = t1
		}

		// add filepaths to list
		if _, found := hostPaths[inst.Host()]; !found {
			hostPaths[inst.Host()] = set.NewStringSet()
		}
		hostPaths[inst.Host()].Insert(fmt.Sprintf("%s/conf/*", inst.Attributes()["deploy_dir"]))
	}

	// build scraper tasks
	for h, t := range hostTasks {
		host := h
		t = t.
			Shell(
				host,
				fmt.Sprintf("%s --config '%s' -f '%s' -t '%s'",
					filepath.Join(task.CheckToolsPathDir, "bin", "scraper"),
					strings.Join(hostPaths[host].Slice(), ","),
					c.ScrapeBegin, c.ScrapeEnd,
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

	t := task.NewBuilder().
		ParallelStep("+ Download necessary tools", false, downloadTasks...).
		ParallelStep("+ Collect host information", false, dryRunTasks...).
		Build()

	ctx := ctxt.New(context.Background(), c.opt.Concurrency)
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
func (c *ConfigCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	switch m.mode {
	case CollectModeTiUP:
		return c.collectForTiUP(m, topo)
	case CollectModeK8s:
		return c.collectForK8s(m, topo)
	}

	return nil
}

// collectForTiUP implements config collecting for tiup-cluster deployed clusters
func (c *ConfigCollectOptions) collectForTiUP(m *Manager, topo *models.TiDBCluster) error {
	rawTopo := topo.Attributes[CollectModeTiUP].(*spec.Specification)

	var (
		collectTasks []*task.StepDisplay
		cleanTasks   []*task.StepDisplay
		queryTasks   []*task.StepDisplay
	)
	ctx := ctxt.New(context.Background(), c.opt.Concurrency)

	uniqueHosts := map[string]int{} // host -> ssh-port

	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	components := topo.Components()
	components = models.FilterComponent(components, roleFilter)
	instances := models.FilterInstance(components, nodeFilter)

	for _, inst := range instances {
		switch inst.Type() {
		case models.ComponentTypeMonitor,
			models.ComponentTypeTiSpark:
			continue
		}

		// ops that applies to each host
		if _, found := uniqueHosts[inst.Host()]; !found {
			uniqueHosts[inst.Host()] = inst.SSHPort()
			t1, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, rawTopo, c.GetBaseOptions().User, *c.opt)
			if err != nil {
				return err
			}
			for _, f := range c.fileStats[inst.Host()] {
				// build checking tasks
				t1 = t1.
					// check for listening ports
					CopyFile(
						f.Target,
						filepath.Join(c.resultDir, inst.Host(), f.Target),
						inst.Host(),
						true,
						c.limit,
					)
				collectTasks = append(
					collectTasks,
					t1.BuildAsStep(fmt.Sprintf("  - Downloading config files from node %s", inst.Host())),
				)
			}
		}

		b, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, rawTopo, c.GetBaseOptions().User, *c.opt)
		if err != nil {
			return err
		}
		t2 := b.
			Rmdir(inst.Host(), task.CheckToolsPathDir).
			BuildAsStep(fmt.Sprintf("  - Cleanup temp files on %s:%d", inst.Host(), inst.SSHPort()))
		cleanTasks = append(cleanTasks, t2)

		// query realtime configs for each instance if supported
		// TODO: support TLS enabled clusters
		if t3 := buildRealtimeConfigCollectingTasks(ctx, inst, c.resultDir, nil); t3 != nil {
			queryTasks = append(queryTasks, t3)
		}
	}

	t := task.NewBuilder().
		ParallelStep("+ Scrap files on nodes", false, collectTasks...).
		ParallelStep("+ Cleanup temp files", false, cleanTasks...).
		ParallelStep("+ Query realtime configs", false, queryTasks...).
		Build()

	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

// collectForK8s implements config collecting for tidb-operator deployed clusters
func (c *ConfigCollectOptions) collectForK8s(_ *Manager, topo *models.TiDBCluster) error {
	var (
		queryTasks []*task.StepDisplay
	)
	ctx := ctxt.New(context.Background(), c.opt.Concurrency)

	/*
		roleFilter := set.NewStringSet(c.opt.Roles...)
		nodeFilter := set.NewStringSet(c.opt.Nodes...)
		components := topo.Components()
		components = models.FilterComponent(components, roleFilter)
		instances := models.FilterInstance(components, nodeFilter)
	*/
	instances := topo.Components()

	for _, inst := range instances {
		switch inst.Type() {
		case models.ComponentTypeMonitor,
			models.ComponentTypeTiSpark:
			continue
		}

		// query realtime configs for each instance if supported
		// TODO: support TLS enabled clusters
		if t3 := buildRealtimeConfigCollectingTasks(ctx, inst, c.resultDir, nil); t3 != nil {
			queryTasks = append(queryTasks, t3)
		}
	}

	t := task.NewBuilder().
		ParallelStep("+ Query realtime configs", false, queryTasks...).
		Build()

	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

type rtConfig struct {
	filename string
	url      string
}

func buildRealtimeConfigCollectingTasks(_ context.Context, inst models.Component, resultDir string, tlsCfg *tls.Config) *task.StepDisplay {
	var configs []rtConfig
	var instDir string
	scheme := "http"

	switch inst.Type() {
	case models.ComponentTypePD:
		configs = append(configs, rtConfig{"config.json", fmt.Sprintf("%s://%s:%d/pd/api/v1/config", scheme, inst.Host(), inst.StatusPort())})
		configs = append(configs, rtConfig{"placement-rule.json", fmt.Sprintf("%s://%s:%d/pd/api/v1/config/placement-rule", scheme, inst.Host(), inst.StatusPort())})
	case models.ComponentTypeTiKV:
		configs = append(configs, rtConfig{"config.json", fmt.Sprintf("%s://%s:%d/config", scheme, inst.Host(), inst.StatusPort())})
	case models.ComponentTypeTiDB:
		configs = append(configs, rtConfig{"config.json", fmt.Sprintf("%s://%s:%d/config", scheme, inst.Host(), inst.StatusPort())})
	case models.ComponentTypeTiFlash:
		configs = append(configs, rtConfig{"config.json", fmt.Sprintf("%s://%s:%d/config", scheme, inst.Host(), inst.StatusPort())})
	default:
		// not supported yet, just ignore
		return nil
	}

	host := inst.Host()
	instDir, ok := inst.Attributes()["deploy_dir"].(string)
	if !ok {
		// for tidb-operator deployed cluster
		instDir = ""
	}
	if pod, ok := inst.Attributes()["pod"].(string); ok {
		host = pod
	}

	t := task.NewBuilder().
		Func(
			fmt.Sprintf("querying %s:%d", host, inst.MainPort()),
			func(ctx context.Context) error {
				c := utils.NewHTTPClient(time.Second*3, tlsCfg)
				for _, config := range configs {
					resp, err := c.Get(ctx, config.url)
					if err != nil {
						fmt.Printf("fail querying %s: %s, continue", config.url, err)
						return nil
					}
					fpath := filepath.Join(resultDir, host, instDir, "conf")
					if err := utils.CreateDir(fpath); err != nil {
						return err
					}
					err = os.WriteFile(
						filepath.Join(resultDir, inst.Host(), instDir, "conf", config.filename),
						resp,
						0644,
					)
					if err != nil {
						return err
					}
				}
				return nil
			},
		).
		BuildAsStep(fmt.Sprintf(
			"  - Querying configs for %s %s:%d",
			inst.Type(),
			inst.Host(),
			inst.MainPort(),
		))

	return t
}
