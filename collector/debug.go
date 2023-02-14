// Copyright 2022 PingCAP, Inc.
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

// DebugCollectOptions are options used collecting debug info
type DebugCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	resultDir string
	fileStats map[string][]CollectStat
	tlsCfg    *tls.Config
}

// Desc implements the Collector interface
func (c *DebugCollectOptions) Desc() string {
	return "Debug info of components"
}

// GetBaseOptions implements the Collector interface
func (c *DebugCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *DebugCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *DebugCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *DebugCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *DebugCollectOptions) Prepare(m *Manager, topo *models.TiDBCluster) (map[string][]CollectStat, error) {
	switch m.mode {
	case CollectModeTiUP:
		return c.prepareForTiUP(m, topo)
	}
	return nil, nil
}

// // prepareForTiUP implements preparation for tiup-cluster deployed clusters
func (c *DebugCollectOptions) prepareForTiUP(_ *Manager, topo *models.TiDBCluster) (map[string][]CollectStat, error) {
	// filter nodes or roles
	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	comps := topo.Components()
	comps = models.FilterComponent(comps, roleFilter)
	instances := models.FilterInstance(comps, nodeFilter)

	for _, inst := range instances {
		switch inst.Type() {
		case models.ComponentTypeTiCDC:
			stat := CollectStat{
				Target: fmt.Sprintf("%s:%d %s debug", inst.Host(), inst.MainPort(), inst.Type()),
				Size:   1024 * 5 * 5,
			}

			c.fileStats[inst.Host()] = append(c.fileStats[inst.Host()], stat)
		}
	}

	return c.fileStats, nil
}

// Collect implements the Collector interface
func (c *DebugCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)

	collecteDebugTasks := []*task.StepDisplay{}

	// filter nodes or roles
	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	comps := topo.Components()
	comps = models.FilterComponent(comps, roleFilter)
	instances := models.FilterInstance(comps, nodeFilter)

	// build tsaks
	for _, inst := range instances {
		if t := buildDebugCollectingTasks(ctx, inst, c); len(t) != 0 {
			collecteDebugTasks = append(collecteDebugTasks, t...)
		}
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Query Debug info", false, collecteDebugTasks...).Build()

	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

type debugConfig struct {
	filepath string
	url      string
}

// buildDebugCollectingTasks build collect debug information tasks
func buildDebugCollectingTasks(ctx context.Context, inst models.Component, c *DebugCollectOptions) []*task.StepDisplay {
	var (
		debugTasks   []*task.StepDisplay
		debugConfigs []debugConfig
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
	case models.ComponentTypeTiCDC:
		// /debug/info
		debugConfigs = append(debugConfigs,
			debugConfig{
				filepath: filepath.Join(c.resultDir, host, instDir, CollectTypeDebug, "info.txt"),
				url:      fmt.Sprintf("%s/debug/info", inst.StatusURL()),
			})

		// /status
		debugConfigs = append(debugConfigs,
			debugConfig{
				filepath: filepath.Join(c.resultDir, host, instDir, CollectTypeDebug, "status.txt"),
				url:      fmt.Sprintf("%s/status", inst.StatusURL()),
			})

		// changefeeds
		debugConfigs = append(debugConfigs,
			debugConfig{
				filepath: filepath.Join(c.resultDir, host, instDir, CollectTypeDebug, "changefeeds.txt"),
				url:      fmt.Sprintf("%s/api/v1/changefeeds", inst.StatusURL()),
			})

		// captures
		debugConfigs = append(debugConfigs,
			debugConfig{
				filepath: filepath.Join(c.resultDir, host, instDir, CollectTypeDebug, "captures.txt"),
				url:      fmt.Sprintf("%s/api/v1/captures", inst.StatusURL()),
			})

		// processors
		debugConfigs = append(debugConfigs,
			debugConfig{
				filepath: filepath.Join(c.resultDir, host, instDir, CollectTypeDebug, "processors.txt"),
				url:      fmt.Sprintf("%s/api/v1/processors", inst.StatusURL()),
			})

	default:
		// not supported yet, just ignore
		return nil
	}

	scheme := "http"
	if c.tlsCfg != nil {
		scheme = "https"
	}

	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)

	t := task.NewBuilder(logger).
		Func(
			fmt.Sprintf("querying %s:%d", host, inst.MainPort()),
			func(ctx context.Context) error {
				c := utils.NewHTTPClient(time.Second*15, c.tlsCfg)
				for _, config := range debugConfigs {
					url := fmt.Sprintf("%s://%s", scheme, config.url)
					err := c.Download(ctx, url, config.filepath)
					if err != nil {
						logger.Warnf("fail querying debug info %s: %s, continue", url, err)
						return err
					}
				}
				return nil
			},
		).
		BuildAsStep(fmt.Sprintf(
			"  - Querying debug info for %s %s:%d",
			inst.Type(),
			inst.Host(),
			inst.MainPort(),
		))
	debugTasks = append(debugTasks, t)

	return debugTasks
}
