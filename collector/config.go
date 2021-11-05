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
func (c *ConfigCollectOptions) Prepare(m *Manager, topo *spec.Specification) (map[string][]CollectStat, error) {
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
		case spec.ComponentPrometheus,
			spec.ComponentGrafana,
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
				t0 := task.NewBuilder().
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
				b, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
				if err != nil {
					return nil, err
				}
				t1 := b.
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
			hostPaths[inst.GetHost()].Insert(fmt.Sprintf("%s/conf/*", inst.DeployDir()))
		}
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
func (c *ConfigCollectOptions) Collect(m *Manager, topo *spec.Specification) error {
	var (
		collectTasks []*task.StepDisplay
		cleanTasks   []*task.StepDisplay
		queryTasks   []*task.StepDisplay
	)
	ctx := ctxt.New(context.Background(), c.opt.Concurrency)

	uniqueHosts := map[string]int{} // host -> ssh-port

	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	components := topo.ComponentsByUpdateOrder()
	components = operator.FilterComponent(components, roleFilter)

	for _, comp := range components {
		switch comp.Name() {
		case spec.ComponentPrometheus,
			spec.ComponentGrafana,
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
			// ops that applies to each host
			if _, found := uniqueHosts[inst.GetHost()]; !found {
				uniqueHosts[inst.GetHost()] = inst.GetSSHPort()
				t1, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
				if err != nil {
					return err
				}
				for _, f := range c.fileStats[inst.GetHost()] {
					// build checking tasks
					t1 = t1.
						// check for listening ports
						CopyFile(
							f.Target,
							filepath.Join(c.resultDir, inst.GetHost(), f.Target),
							inst.GetHost(),
							true,
							c.limit,
						)
					collectTasks = append(
						collectTasks,
						t1.BuildAsStep(fmt.Sprintf("  - Downloading config files from node %s", inst.GetHost())),
					)
				}
			}

			b, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
			if err != nil {
				return err
			}
			t2 := b.
				Rmdir(inst.GetHost(), task.CheckToolsPathDir).
				BuildAsStep(fmt.Sprintf("  - Cleanup temp files on %s:%d", inst.GetHost(), inst.GetSSHPort()))
			cleanTasks = append(cleanTasks, t2)

			// query realtime configs for each instance if supported
			// TODO: support TLS enabled clusters
			if t3 := buildRealtimeConfigCollectingTasks(ctx, inst, c.resultDir, nil); t3 != nil {
				queryTasks = append(queryTasks, t3)
			}
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

type rtConfig struct {
	filename string
	url      string
}

func buildRealtimeConfigCollectingTasks(_ context.Context, inst spec.Instance, resultDir string, tlsCfg *tls.Config) *task.StepDisplay {
	var configs []rtConfig
	var instDir string
	scheme := "http"

	switch inst.ComponentName() {
	case spec.ComponentPD:
		i := inst.(*spec.PDInstance).BaseInstance.InstanceSpec.(*spec.PDSpec)
		configs = append(configs, rtConfig{"config.json", fmt.Sprintf("%s://%s:%d/pd/api/v1/config", scheme, i.Host, i.ClientPort)})
		configs = append(configs, rtConfig{"placement-rule.json", fmt.Sprintf("%s://%s:%d/pd/api/v1/config/placement-rule", scheme, i.Host, i.ClientPort)})
	case spec.ComponentTiKV:
		i := inst.(*spec.TiKVInstance).BaseInstance.InstanceSpec.(*spec.TiKVSpec)
		configs = append(configs, rtConfig{"config.json", fmt.Sprintf("%s://%s:%d/config", scheme, i.Host, i.StatusPort)})
	case spec.ComponentTiDB:
		i := inst.(*spec.TiDBInstance).BaseInstance.InstanceSpec.(*spec.TiDBSpec)
		configs = append(configs, rtConfig{"config.json", fmt.Sprintf("%s://%s:%d/config", scheme, i.Host, i.StatusPort)})
	case spec.ComponentTiFlash:
		i := inst.(*spec.TiFlashInstance).BaseInstance.InstanceSpec.(*spec.TiFlashSpec)
		configs = append(configs, rtConfig{"config.json", fmt.Sprintf("%s://%s:%d/config", scheme, i.Host, i.FlashProxyStatusPort)})
	default:
		// not supported yet, just ignore
		return nil
	}

	instDir = inst.DeployDir()

	t := task.NewBuilder().
		Func(
			fmt.Sprintf("querying %s:%d", inst.GetHost(), inst.GetMainPort()),
			func(ctx context.Context) error {
				c := utils.NewHTTPClient(time.Second*3, tlsCfg)
				for _, config := range configs {
					resp, err := c.Get(ctx, config.url)
					if err != nil {
						return err
					}
					err = os.WriteFile(
						filepath.Join(resultDir, inst.GetHost(), instDir, "conf", config.filename),
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
			inst.ComponentName(),
			inst.GetHost(),
			inst.GetMainPort(),
		))

	return t
}
