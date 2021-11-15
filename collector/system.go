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
	"os"
	"path/filepath"

	"github.com/joomcode/errorx"
	jsoniter "github.com/json-iterator/go"
	"github.com/pingcap/diag/pkg/models"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tidb-insight/collector/insight"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/utils"
)

// SystemCollectOptions are options used collecting system information
type SystemCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	resultDir string
}

// Desc implements the Collector interface
func (c *SystemCollectOptions) Desc() string {
	return "basic system information of servers"
}

// GetBaseOptions implements the Collector interface
func (c *SystemCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *SystemCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *SystemCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *SystemCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *SystemCollectOptions) Prepare(_ *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *SystemCollectOptions) Collect(m *Manager, cls *models.TiDBCluster) error {
	if m.mode != CollectModeTiUP {
		return nil
	}

	topo := cls.Attributes[CollectModeTiUP].(*spec.Specification)
	var (
		collectInsightTasks []*task.StepDisplay
		checkSysTasks       []*task.StepDisplay
		cleanTasks          []*task.StepDisplay
		downloadTasks       []*task.StepDisplay
	)
	insightVer := spec.TiDBComponentVersion(spec.ComponentCheckCollector, "")

	uniqueHosts := map[string]int{}             // host -> ssh-port
	uniqueArchList := make(map[string]struct{}) // map["os-arch"]{}

	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	components := topo.ComponentsByUpdateOrder()
	components = operator.FilterComponent(components, roleFilter)

	for _, comp := range components {
		instances := operator.FilterInstance(comp.Instances(), nodeFilter)
		if len(instances) < 1 {
			continue
		}

		for _, inst := range instances {
			archKey := fmt.Sprintf("%s-%s", inst.OS(), inst.Arch())
			if _, found := uniqueArchList[archKey]; !found {
				uniqueArchList[archKey] = struct{}{}
				t0 := task.NewBuilder(m.DisplayMode).
					Download(
						spec.ComponentCheckCollector,
						inst.OS(),
						inst.Arch(),
						insightVer,
					).
					BuildAsStep(fmt.Sprintf("  - Downloading check tools for %s/%s", inst.OS(), inst.Arch()))
				downloadTasks = append(downloadTasks, t0)
			}

			// checks that applies to each host
			if _, found := uniqueHosts[inst.GetHost()]; !found {
				uniqueHosts[inst.GetHost()] = inst.GetSSHPort()
				// build system info collecting tasks
				b1, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
				if err != nil {
					return err
				}
				t1 := b1.
					Mkdir(c.GetBaseOptions().User, inst.GetHost(), filepath.Join(task.CheckToolsPathDir, "bin")).
					CopyComponent(
						spec.ComponentCheckCollector,
						inst.OS(),
						inst.Arch(),
						insightVer,
						"", // use default srcPath
						inst.GetHost(),
						task.CheckToolsPathDir,
					).
					Shell(
						inst.GetHost(),
						fmt.Sprintf("%s %s",
							filepath.Join(task.CheckToolsPathDir, "bin", "insight"),
							"--syscfg --dmesg", // enable collecting of extra info
						),
						"",
						true,
					).
					Func(
						inst.GetHost(),
						func(ctx context.Context) error {
							return saveInsightOutput(ctx, inst.GetHost(), c.resultDir)
						},
					).
					BuildAsStep(fmt.Sprintf("  - Getting system info of %s:%d", inst.GetHost(), inst.GetSSHPort()))
				collectInsightTasks = append(collectInsightTasks, t1)

				// build checking tasks
				t2 := task.NewBuilder(m.DisplayMode).
					// check for listening ports
					Shell(
						inst.GetHost(),
						"ss -lanp",
						"",
						false,
					).
					Func(
						inst.GetHost(),
						func(ctx context.Context) error {
							return saveRawOutput(ctx, inst.GetHost(), c.resultDir, "ss.txt")
						},
					)
				checkSysTasks = append(
					checkSysTasks,
					t2.BuildAsStep(fmt.Sprintf("  - Collecting system info of node %s", inst.GetHost())),
				)
			}

			b3, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
			if err != nil {
				return err
			}
			t3 := b3.
				Rmdir(inst.GetHost(), task.CheckToolsPathDir).
				BuildAsStep(fmt.Sprintf("  - Cleanup temp files on %s:%d", inst.GetHost(), inst.GetSSHPort()))
			cleanTasks = append(cleanTasks, t3)
		}
	}

	t := task.NewBuilder(m.DisplayMode).
		ParallelStep("+ Download necessary tools", false, downloadTasks...).
		ParallelStep("+ Collect host information", false, collectInsightTasks...).
		ParallelStep("+ Collect system information", false, checkSysTasks...).
		ParallelStep("+ Cleanup temp files", false, cleanTasks...).
		Build()

	ctx := ctxt.New(context.Background(), c.opt.Concurrency)
	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

func saveOutput(data []byte, fname string) error {
	dir := filepath.Dir(fname)
	if err := utils.CreateDir(dir); err != nil {
		return err
	}

	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}
	return nil
}

func saveRawOutput(ctx context.Context, host, dir, fname string) error {
	stdout, stderr, _ := ctxt.GetInner(ctx).GetOutputs(host)
	if len(stderr) > 0 {
		if err := saveOutput(stderr, filepath.Join(dir, fmt.Sprintf("%s.stderr", fname))); err != nil {
			return err
		}
	}
	return saveOutput(stdout, filepath.Join(dir, host, fname))
}

func saveInsightOutput(ctx context.Context, host, dir string) error {
	stdout, stderr, _ := ctxt.GetInner(ctx).GetOutputs(host)
	if len(stderr) > 0 {
		if err := saveOutput(stderr, filepath.Join(dir, host, "insight.stderr")); err != nil {
			return err
		}
	}

	var info insight.InsightInfo
	if err := jsoniter.Unmarshal(stdout, &info); err != nil {
		// save output directly on parsing errors
		return saveOutput(stdout, filepath.Join(dir, host, "insight.json"))
	}

	// save limits and kernel parameters
	seclim := make([]byte, 0)
	sysctl := make([]byte, 0)
	for _, item := range info.SysConfig.SecLimit {
		seclim = append(seclim,
			[]byte(fmt.Sprintf("%s\t%s\t%s\t%d\n", item.Domain, item.Type, item.Item, item.Value))...,
		)
	}
	if err := saveOutput(seclim, filepath.Join(dir, host, "limits.conf")); err != nil {
		return err
	}
	for k, v := range info.SysConfig.SysCtl {
		sysctl = append(sysctl,
			[]byte(fmt.Sprintf("%s = %s\n", k, v))...,
		)
	}
	if err := saveOutput(sysctl, filepath.Join(dir, host, "sysctl.conf")); err != nil {
		return err
	}

	// save kernel log
	dmesg := make([]byte, 0)
	for _, item := range info.DMesg {
		dmesg = append(dmesg, []byte(fmt.Sprintln(item))...)
	}
	if err := saveOutput(dmesg, filepath.Join(dir, host, "dmesg.log")); err != nil {
		return err
	}

	return saveOutput(stdout, filepath.Join(dir, host, "insight.json"))
}
