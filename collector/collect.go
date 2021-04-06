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
	"github.com/pingcap/tiup/pkg/cliutil"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	"github.com/pingcap/tiup/pkg/set"
)

// CollectOptions contains the options for check command
type CollectOptions struct {
	User         string // username to login to the SSH server
	IdentityFile string // path to the private key file
	UsePassword  bool   // use password instead of identity file for ssh connection
	ScrapeBegin  string // start timepoint when collecting metrics and logs
	ScrapeEnd    string // stop timepoint when collecting metrics and logs
}

// CollectClusterInfo collects information and metrics from a tidb cluster
func (m *Manager) CollectClusterInfo(clusterName string, opt CollectOptions, gOpt operator.Options) error {
	var topo spec.Specification

	exist, err := m.specManager.Exist(clusterName)
	if err != nil {
		return err
	}
	if !exist {
		return perrs.Errorf("cluster %s does not exist", clusterName)
	}

	metadata, err := spec.ClusterMetadata(clusterName)
	if err != nil {
		return err
	}
	opt.User = metadata.User
	opt.IdentityFile = m.specManager.Path(clusterName, "ssh", "id_rsa")
	topo = *metadata.Topology

	// prepare output dir of collected data
	resultDir := m.specManager.Path(clusterName, "collector", m.session)
	if err := os.MkdirAll(resultDir, 0755); err != nil {
		return err
	}

	// save the topology
	yamlMeta, err := os.ReadFile(m.specManager.Path(clusterName, "meta.yaml"))
	if err != nil {
		return err
	}
	fm, err := os.Create(filepath.Join(resultDir, "meta.yaml"))
	if err != nil {
		return err
	}
	defer fm.Close()
	if _, err := fm.Write(yamlMeta); err != nil {
		return err
	}

	// collect data from remote servers
	var sshConnProps *cliutil.SSHConnectionProps = &cliutil.SSHConnectionProps{}
	if gOpt.SSHType != executor.SSHTypeNone {
		var err error
		if sshConnProps, err = cliutil.ReadIdentityFileOrPassword(opt.IdentityFile, opt.UsePassword); err != nil {
			return err
		}
	}

	if err := collectSystemInfo(sshConnProps, &topo, &gOpt, &opt, resultDir); err != nil {
		return err
	}

	fmt.Println("Collecting alert lists from Prometheus node...")
	if err := collectAlerts(&topo, resultDir); err != nil {
		return err
	}

	fmt.Println("Collecting metrics from Prometheus node...")
	if err := collectMetrics(&topo, &opt, resultDir); err != nil {
		return err
	}

	fmt.Printf("Collected data are stored in %s\n", resultDir)
	return nil
}

// collectSystemInfo gathers many information of the deploy server
func collectSystemInfo(
	s *cliutil.SSHConnectionProps,
	topo *spec.Specification,
	gOpt *operator.Options,
	opt *CollectOptions,
	resultDir string,
) error {
	var (
		collectInsightTasks []*task.StepDisplay
		checkSysTasks       []*task.StepDisplay
		cleanTasks          []*task.StepDisplay
		downloadTasks       []*task.StepDisplay
	)
	insightVer := spec.TiDBComponentVersion(spec.ComponentCheckCollector, "")

	uniqueHosts := map[string]int{}             // host -> ssh-port
	uniqueArchList := make(map[string]struct{}) // map["os-arch"]{}

	roleFilter := set.NewStringSet(gOpt.Roles...)
	nodeFilter := set.NewStringSet(gOpt.Nodes...)
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
						opt.User,
						s.Password,
						s.IdentityFile,
						s.IdentityFilePassphrase,
						gOpt.SSHTimeout,
						gOpt.SSHType,
						topo.GlobalOptions.SSHType,
					).
					Mkdir(opt.User, inst.GetHost(), filepath.Join(task.CheckToolsPathDir, "bin")).
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
							return saveOutput(ctx, inst.GetHost(), resultDir, "insight.json")
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
							return saveOutput(ctx, inst.GetHost(), resultDir, "ss.txt")
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
					opt.User,
					s.Password,
					s.IdentityFile,
					s.IdentityFilePassphrase,
					gOpt.SSHTimeout,
					gOpt.SSHType,
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
