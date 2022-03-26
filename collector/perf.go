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
	httpjob "github.com/pingcap/diag/pkg/http"
	"github.com/pingcap/diag/pkg/models"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/set"
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
				Target: fmt.Sprintf("%s:%d %s perf", inst.Host(), inst.MainPort(), inst.Type()),
				Size:   fsize,
			}

			c.fileStats[inst.Host()] = append(c.fileStats[inst.Host()], stat)

		case models.ComponentTypeTiKV, models.ComponentTypeTiFlash:
			// cpu profile
			c.fileStats[inst.Host()] = append(c.fileStats[inst.Host()], CollectStat{
				Target: fmt.Sprintf("%s:%d %s perf", inst.Host(), inst.MainPort(), inst.Type()),
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

// buildPerfCollectingTasks build collect profile information tasks
func buildPerfCollectingTasks(ctx context.Context, inst models.Component, c *PerfCollectOptions) []*task.StepDisplay {
	var (
		perfInfoTasks []*task.StepDisplay
		httpJobs      []httpjob.HttpCollectJob
	)

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
		httpJobs = append(httpJobs,
			*httpjob.NewHttpJob(
				filepath.Join(c.resultDir, host, instDir, CollectTypePerf, "cpu_profile.proto"),
				fmt.Sprintf("%s/debug/pprof/profile?seconds=%d", inst.StatusURL(), c.duration),
				httpjob.WithTlsCfg(c.tlsCfg),
				httpjob.WithTimeOut(time.Second*time.Duration(c.duration+3)),
			),
		)

		// mem Heap
		httpJobs = append(httpJobs,
			*httpjob.NewHttpJob(
				filepath.Join(c.resultDir, host, instDir, CollectTypePerf, "mem_heap.proto"),
				fmt.Sprintf("%s/debug/pprof/heap", inst.StatusURL()),
				httpjob.WithTlsCfg(c.tlsCfg),
			),
		)

		// Goroutine
		httpJobs = append(httpJobs,
			*httpjob.NewHttpJob(
				filepath.Join(c.resultDir, host, instDir, CollectTypePerf, "goroutine.txt"),
				fmt.Sprintf("%s/debug/pprof/goroutine?debug=1", inst.StatusURL()),
				httpjob.WithTlsCfg(c.tlsCfg),
			),
		)

		// mutex
		httpJobs = append(httpJobs,
			*httpjob.NewHttpJob(
				filepath.Join(c.resultDir, host, instDir, CollectTypePerf, "mutex.txt"),
				fmt.Sprintf("%s/debug/pprof/mutex?debug=1", inst.StatusURL()),
				httpjob.WithTlsCfg(c.tlsCfg),
			),
		)

	case models.ComponentTypeTiKV, models.ComponentTypeTiFlash:
		// cpu profile
		httpJobs = append(httpJobs,
			*httpjob.NewHttpJob(
				filepath.Join(c.resultDir, host, instDir, CollectTypePerf, "cpu_profile.proto"),
				fmt.Sprintf("%s/debug/pprof/profile?seconds=%d", inst.StatusURL(), c.duration),
				httpjob.WithTlsCfg(c.tlsCfg),
				httpjob.WithTimeOut(time.Second*time.Duration(c.duration+3)),
				httpjob.WithHeader(map[string]string{"Content-Type": "application/protobuf"}),
			),
		)
	default:
		// not supported yet, just ignore
		return nil
	}

	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)
	t := task.NewBuilder(logger).
		Func(
			fmt.Sprintf("querying %s:%d", host, inst.MainPort()),
			func(ctx context.Context) error {
				for _, job := range httpJobs {
					err := job.Do(ctx)
					if err != nil {
						return err
					}
				}
				return nil
			},
		).
		BuildAsStep(fmt.Sprintf(
			"  - Querying profile info for %s %s:%d",
			inst.Type(),
			inst.Host(),
			inst.MainPort(),
		))
	perfInfoTasks = append(perfInfoTasks, t)

	return perfInfoTasks
}
