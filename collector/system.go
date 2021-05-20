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
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	"github.com/pingcap/tiup/pkg/set"
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

// Collect implements the Collector interface
func (c *SystemCollectOptions) Collect(topo *spec.Specification) error {
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
				t0 := task.NewBuilder().
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
				t1 := task.NewBuilder().
					RootSSH(
						inst.GetHost(),
						inst.GetSSHPort(),
						c.BaseOptions.User,
						c.SSH.Password,
						c.SSH.IdentityFile,
						c.SSH.IdentityFilePassphrase,
						c.opt.SSHTimeout,
						c.opt.SSHType,
						topo.GlobalOptions.SSHType,
					).
					Mkdir(c.BaseOptions.User, inst.GetHost(), filepath.Join(task.CheckToolsPathDir, "bin")).
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
						filepath.Join(task.CheckToolsPathDir, "bin", "insight"),
						"",
						false,
					).
					Func(
						inst.GetHost(),
						func(ctx context.Context) error {
							return saveOutput(ctx, inst.GetHost(), c.resultDir, "insight.json")
						},
					).
					BuildAsStep(fmt.Sprintf("  - Getting system info of %s:%d", inst.GetHost(), inst.GetSSHPort()))
				collectInsightTasks = append(collectInsightTasks, t1)

				// build checking tasks
				t2 := task.NewBuilder().
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
							return saveOutput(ctx, inst.GetHost(), c.resultDir, "ss.txt")
						},
					).
					// gather kernel configs
					Shell(
						inst.GetHost(),
						"sysctl -a",
						"",
						true,
					).
					Func(
						inst.GetHost(),
						func(ctx context.Context) error {
							return saveOutput(ctx, inst.GetHost(), c.resultDir, "sysctl.txt")
						},
					)
				checkSysTasks = append(
					checkSysTasks,
					t2.BuildAsStep(fmt.Sprintf("  - Collecting system info of node %s", inst.GetHost())),
				)
			}

			t3 := task.NewBuilder().
				RootSSH(
					inst.GetHost(),
					inst.GetSSHPort(),
					c.BaseOptions.User,
					c.SSH.Password,
					c.SSH.IdentityFile,
					c.SSH.IdentityFilePassphrase,
					c.opt.SSHTimeout,
					c.opt.SSHType,
					topo.GlobalOptions.SSHType,
				).
				Rmdir(inst.GetHost(), task.CheckToolsPathDir).
				BuildAsStep(fmt.Sprintf("  - Cleanup temp files on %s:%d", inst.GetHost(), inst.GetSSHPort()))
			cleanTasks = append(cleanTasks, t3)
		}
	}

	t := task.NewBuilder().
		ParallelStep("+ Download necessary tools", false, downloadTasks...).
		ParallelStep("+ Collect host information", false, collectInsightTasks...).
		ParallelStep("+ Collect system information", false, checkSysTasks...).
		ParallelStep("+ Cleanup temp files", false, cleanTasks...).
		Build()

	ctx := ctxt.New(context.Background())
	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

func saveOutput(ctx context.Context, host, dir, fname string) error {
	stdout, stderr, _ := ctxt.GetInner(ctx).GetOutputs(host)

	fo, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s.%s", "stdout", fname)))
	if err != nil {
		return err
	}
	defer fo.Close()
	fe, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s.%s", "stderr", fname)))
	if err != nil {
		return err
	}
	defer fe.Close()

	if _, err := fo.Write(stdout); err != nil {
		return err
	}
	if _, err := fe.Write(stderr); err != nil {
		return err
	}

	return nil
}
