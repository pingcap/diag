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
	"time"

	"github.com/joomcode/errorx"
	"github.com/pingcap/diag/pkg/models"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/utils"
)

// PerfCollectOptions are options used collecting pref info
type PerfCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	limit     int
	duration  int // scp rate limit
	resultDir string
	fileStats map[string][]CollectStat
	compress  bool
}

// Desc implements the Collector interface
func (c *PerfCollectOptions) Desc() string {
	return "Pref info of components"
}

// GetBaseOptions implements the Collector interface
func (c *PerfCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *PerfCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *PerfCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *PerfCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *PerfCollectOptions) Prepare(_ *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *PerfCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)

	collectePerfTasks := []*task.StepDisplay{}

	// filter nodes or roles
	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	comps := topo.Components()
	comps = models.FilterComponent(comps, roleFilter)
	instances := models.FilterInstance(comps, nodeFilter)

	// build tsaks
	for _, inst := range instances {

		// TODO: support TLS enabled clusters
		if t := buildPerfCollectingTasks(ctx, inst, c.resultDir, c.duration, nil); len(t) != 0 {
			collectePerfTasks = append(collectePerfTasks, t...)
		}
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Query profile info", false, collectePerfTasks...).Build()

	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

// perfInfo  profile information
type perfInfo struct {
	filename string
	perfType string
	url      string
	header   map[string]string
	timeout  time.Duration
	proto    bool
}

// http://172.16.7.147:2379

func buildPerfCollectingTasks(ctx context.Context, inst models.Component, resultDir string, duration int, tlsCfg *tls.Config) []*task.StepDisplay {
	var (
		perfInfoTasks []*task.StepDisplay
		perfInfos     []perfInfo
	)
	scheme := "http"
	if tlsCfg != nil {
		scheme = "https"
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

	switch inst.Type() {
	case models.ComponentTypeTiDB, models.ComponentTypePD:
		// cpu profile
		perfInfos = append(perfInfos,
			perfInfo{
				filename: "cpu_profile.proto",
				perfType: "cpu_profile",
				url:      fmt.Sprintf("%s:%d/debug/pprof/profile?seconds=%d", host, inst.StatusPort(), duration),
				timeout:  time.Second * time.Duration(duration+3),
				proto:    true,
			})
		// mem Heap
		perfInfos = append(perfInfos,
			perfInfo{
				filename: "mem_heap.proto",
				perfType: "mem_heap",
				url:      fmt.Sprintf("%s:%d/debug/pprof/heap", host, inst.StatusPort()),
				timeout:  time.Second * 3,
				proto:    true,
			})
		// Goroutine
		perfInfos = append(perfInfos,
			perfInfo{
				filename: "goroutine.txt",
				perfType: "goroutine",
				url:      fmt.Sprintf("%s:%d/debug/pprof/goroutine?debug=1", host, inst.StatusPort()),
				timeout:  time.Second * 3,
			})
		// mutex
		perfInfos = append(perfInfos,
			perfInfo{
				filename: "mutex.txt",
				perfType: "mutex",
				url:      fmt.Sprintf("%s:%d/debug/pprof/mutex?debug=1", host, inst.StatusPort()),
				timeout:  time.Second * 3,
			})
	case models.ComponentTypeTiKV, models.ComponentTypeTiFlash:
		// cpu profile
		perfInfos = append(perfInfos,
			perfInfo{
				filename: "cpu_profile.proto",
				perfType: "cpu_profile",
				header:   map[string]string{"Content-Type": "application/protobuf"},
				url:      fmt.Sprintf("%s:%d/debug/pprof/profile?seconds=%d", host, inst.StatusPort(), duration),
				timeout:  time.Second * time.Duration(duration+3),
				proto:    true,
			})
	default:
		// not supported yet, just ignore
		return nil
	}

	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)
	for _, perfInfo := range perfInfos {
		perfInfo := perfInfo
		t := task.NewBuilder(logger).
			Func(
				fmt.Sprintf("querying %s %s:%d", perfInfo.perfType, host, inst.StatusPort()),
				func(ctx context.Context) error {
					c := utils.NewHTTPClient(perfInfo.timeout, tlsCfg)

					if perfInfo.header != nil {
						for k, v := range perfInfo.header {
							c.SetRequestHeader(k, v)
						}
					}

					url := fmt.Sprintf("%s://%s", scheme, perfInfo.url)
					fpath := filepath.Join(resultDir, host, instDir, "perf")
					fFile := filepath.Join(resultDir, host, instDir, "perf", perfInfo.filename)
					if err := utils.CreateDir(fpath); err != nil {
						return err
					}

					if perfInfo.proto {
						err := c.Download(ctx, url, fFile)
						if err != nil {
							logger.Warnf("fail querying %s: %s, continue", url, err)
							return err
						}

					} else {
						resp, err := c.Get(ctx, url)
						if err != nil {
							logger.Warnf("fail querying %s: %s, continue", url, err)
							return err
						}

						err = os.WriteFile(
							fFile,
							resp,
							0644,
						)
						if err != nil {
							logger.Warnf("fail querying %s: %s, continue", url, err)
							return err
						}

					}

					return nil
				},
			).
			BuildAsStep(fmt.Sprintf(
				"  - Querying %s for %s %s:%d",
				perfInfo.perfType,
				inst.Type(),
				inst.Host(),
				inst.StatusPort(),
			))
		perfInfoTasks = append(perfInfoTasks, t)
	}

	return perfInfoTasks
}
