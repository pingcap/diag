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
	duration  int               //seconds: profile time(s), default is 30s.
	resultDir string
	fileStats map[string][]CollectStat
	tlsCfg    *tls.Config
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
func (c *PerfCollectOptions) Prepare(m *Manager, topo *models.TiDBCluster) (map[string][]CollectStat, error) {

	switch m.mode {
	case CollectModeTiUP:
		return c.prepareForTiUP(m, topo)
	}
	return nil, nil
}

// // prepareForTiUP implements preparation for tiup-cluster deployed clusters
func (c *PerfCollectOptions) prepareForTiUP(_ *Manager, topo *models.TiDBCluster) (map[string][]CollectStat, error) {
	// filter nodes or roles
	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	comps := topo.Components()
	comps = models.FilterComponent(comps, roleFilter)
	instances := models.FilterInstance(comps, nodeFilter)

	for _, inst := range instances {
		switch inst.Type() {
		case models.ComponentTypeTiDB, models.ComponentTypePD, models.ComponentTypeTiCDC:

			// cpu profile
			fsize := (6 * 1024) * int64(c.duration)

			// mem Heap
			fsize = fsize + 500*1024

			// Goroutine
			fsize = fsize + 100*1024

			// mutex
			fsize = fsize + 500*1024

			if inst.Type() == models.ComponentTypePD {
				fsize = fsize + 800*1024
			}

			stat := CollectStat{
				Target: fmt.Sprintf("%s:%d", inst.Host(), inst.MainPort()),
				Size:   fsize,
			}

			c.fileStats[inst.Host()] = append(c.fileStats[inst.Host()], stat)

		case models.ComponentTypeTiKV, models.ComponentTypeTiFlash:
			// cpu profile
			c.fileStats[inst.Host()] = append(c.fileStats[inst.Host()], CollectStat{
				Target: fmt.Sprintf("%s:%d", inst.Host(), inst.MainPort()),
				Size:   (18 * 1024) * int64(c.duration),
			})
		}

	}

	return c.fileStats, nil
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

		if t := buildPerfCollectingTasks(ctx, inst, c); len(t) != 0 {
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
}

// buildPerfCollectingTasks build collect profile information tasks
func buildPerfCollectingTasks(ctx context.Context, inst models.Component, c *PerfCollectOptions) []*task.StepDisplay {
	var (
		perfInfoTasks []*task.StepDisplay
		perfInfos     []perfInfo
	)
	scheme := "http"
	if c.tlsCfg != nil {
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
	case models.ComponentTypeTiDB, models.ComponentTypePD, models.ComponentTypeTiCDC:
		// cpu profile
		perfInfos = append(perfInfos,
			perfInfo{
				filename: "cpu_profile.proto",
				perfType: "cpu_profile",
				url:      fmt.Sprintf("%s/debug/pprof/profile?seconds=%d", inst.StatusURL(), c.duration),
				timeout:  time.Second * time.Duration(c.duration+3),
			})
		// mem Heap
		perfInfos = append(perfInfos,
			perfInfo{
				filename: "mem_heap.proto",
				perfType: "mem_heap",
				url:      fmt.Sprintf("%s/debug/pprof/heap", inst.StatusURL()),
				timeout:  time.Second * 3,
			})
		// Goroutine
		perfInfos = append(perfInfos,
			perfInfo{
				filename: "goroutine.txt",
				perfType: "goroutine",
				url:      fmt.Sprintf("%s/debug/pprof/goroutine?debug=1", inst.StatusURL()),
				timeout:  time.Second * 3,
			})
		// mutex
		perfInfos = append(perfInfos,
			perfInfo{
				filename: "mutex.txt",
				perfType: "mutex",
				url:      fmt.Sprintf("%s/debug/pprof/mutex?debug=1", inst.StatusURL()),
				timeout:  time.Second * 3,
			})
	case models.ComponentTypeTiKV, models.ComponentTypeTiFlash:
		// cpu profile
		perfInfos = append(perfInfos,
			perfInfo{
				filename: "cpu_profile.proto",
				perfType: "cpu_profile",
				header:   map[string]string{"Content-Type": "application/protobuf"},
				url:      fmt.Sprintf("%s/debug/pprof/profile?seconds=%d", inst.StatusURL(), c.duration),
				timeout:  time.Second * time.Duration(c.duration+3),
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
				fmt.Sprintf("querying %s %s:%d", perfInfo.perfType, host, inst.MainPort()),
				func(ctx context.Context) error {
					httpClient := utils.NewHTTPClient(perfInfo.timeout, c.tlsCfg)

					if perfInfo.header != nil {
						for k, v := range perfInfo.header {
							httpClient.SetRequestHeader(k, v)
						}
					}

					url := fmt.Sprintf("%s://%s", scheme, perfInfo.url)
					fpath := filepath.Join(c.resultDir, host, instDir, "perf")
					fFile := filepath.Join(c.resultDir, host, instDir, "perf", perfInfo.filename)
					if err := utils.CreateDir(fpath); err != nil {
						return err
					}

					err := httpClient.Download(ctx, url, fFile)
					if err != nil {
						logger.Warnf("fail querying %s: %s, continue", url, err)
						return err
					}

					return nil
				},
			).
			BuildAsStep(fmt.Sprintf(
				"  - Querying %s for %s %s:%d",
				perfInfo.perfType,
				inst.Type(),
				inst.Host(),
				inst.MainPort(),
			))
		perfInfoTasks = append(perfInfoTasks, t)
	}

	return perfInfoTasks
}
