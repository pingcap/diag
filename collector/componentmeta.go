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
	"os"
	"path/filepath"

	"github.com/joomcode/errorx"
	"github.com/pingcap/diag/pkg/models"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/set"
	tiuputils "github.com/pingcap/tiup/pkg/utils"
)

// ComponentMetaCollectOptions are options used collecting component metadata
type ComponentMetaCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli  
	resultDir string
	fileStats map[string][]CollectStat
	tlsCfg    *tls.Config
	topo      *models.TiDBCluster
}

// Desc implements the Collector interface
func (c *ComponentMetaCollectOptions) Desc() string {
	return "Metadata of components"
}

// GetBaseOptions implements the Collector interface
func (c *ComponentMetaCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *ComponentMetaCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *ComponentMetaCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *ComponentMetaCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *ComponentMetaCollectOptions) Prepare(m *Manager, topo *models.TiDBCluster) (map[string][]CollectStat, error) {

	if m.mode != CollectModeTiUP {
		return nil, nil
	}

	// filter nodes or roles
	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	comps := topo.Components()
	comps = models.FilterComponent(comps, roleFilter)
	instances := models.FilterInstance(comps, nodeFilter)

	for _, inst := range instances {
		switch inst.Type() {
		case models.ComponentTypeTiCDC:
			c.fileStats[inst.Host()] = append(c.fileStats[inst.Host()], CollectStat{
				Target: fmt.Sprintf("%s:%d %s component_meta", inst.Host(), inst.MainPort(), inst.Type()),
				Size:   (10 * 1024),
			})
		}
	}

	return c.fileStats, nil
}

// Collect implements the Collector interface
func (c *ComponentMetaCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)

	collecteMetaDateTasks := []*task.StepDisplay{}

	// filter nodes or roles
	roleFilter := set.NewStringSet(c.opt.Roles...)
	nodeFilter := set.NewStringSet(c.opt.Nodes...)
	comps := topo.Components()
	comps = models.FilterComponent(comps, roleFilter)
	instances := models.FilterInstance(comps, nodeFilter)

	// build tsaks
	for _, inst := range instances {

		if t := buildMateCollectingTasks(ctx, c, inst); len(t) != 0 {
			collecteMetaDateTasks = append(collecteMetaDateTasks, t...)
		}
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Collect component metadata", false, collecteMetaDateTasks...).Build()

	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

// buildMateCollectingTasks build collect matedata information tasks
func buildMateCollectingTasks(ctx context.Context, c *ComponentMetaCollectOptions, inst models.Component) []*task.StepDisplay {

	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)

	switch inst.Type() {
	case models.ComponentTypeTiCDC:
		tasks, err := buildTiCDCMateCollectTask(ctx, c, inst)
		if err != nil {
			logger.Warnf("fail collect TiCDC component matedata: %s, continue", err)
		}
		return tasks
	default:
		// not supported yet, just ignore
		return nil
	}
}

func buildTiCDCMateCollectTask(ctx context.Context, c *ComponentMetaCollectOptions, inst models.Component) ([]*task.StepDisplay, error) {

	var debugTasks []*task.StepDisplay
	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)

	host := inst.Host()
	instDir, ok := inst.Attributes()["deploy_dir"].(string)
	if !ok {
		// for tidb-operator deployed cluster
		instDir = ""
	}
	if pod, ok := inst.Attributes()["pod"].(string); ok {
		host = pod
	}

	t := task.NewBuilder(logger).
		Func(
			fmt.Sprintf("collect metadata for %s", inst.Type()),
			func(ctx context.Context) error {
				kvs, err := c.topo.GetAllCDCInfo(ctx, c.tlsCfg)
				if err != nil {
					return err
				}
				dir := filepath.Join(c.resultDir, host, instDir, "component_meta")
				if tiuputils.IsNotExist(dir) {
					err := os.MkdirAll(dir, 0755)
					if err != nil {
						return err
					}
				}

				f, err := os.OpenFile(filepath.Join(dir, "metadata.txt"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				if err != nil {
					return err
				}
				defer f.Close()

				for _, kv := range kvs {
					_, err = f.WriteString(fmt.Sprintf("Key: %s, Value: %s\n", string(kv.Key), string(kv.Value)))
					if err != nil {
						logger.Warnf("fail collect TiCDC matedata: %s, continue", err)
					}
				}

				return nil
			},
		).
		BuildAsStep(fmt.Sprintf(
			"  - Querying component metadata for %s %s:%d",
			inst.Type(),
			inst.Host(),
			inst.MainPort(),
		))
	debugTasks = append(debugTasks, t)
	return debugTasks, nil

}
