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
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/diag/scraper"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
)

// AuditLogCollectOptions are options used collecting component logs
type AuditLogCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	resultDir string
	fileStats map[string][]CollectStat
	topoType  string // cluster or dm
}

// Desc implements the Collector interface
func (c *AuditLogCollectOptions) Desc() string {
	return fmt.Sprintf("%s audit logs of components", c.topoType)

}

// GetBaseOptions implements the Collector interface
func (c *AuditLogCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *AuditLogCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *AuditLogCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *AuditLogCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *AuditLogCollectOptions) Prepare(m *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	if m.mode != CollectModeTiUP {
		return nil, nil
	}

	t := task.NewBuilder(m.logger).
		Func(
			"collect audit log information",
			func(ctx context.Context) error {
				files, err := auditLogStat(spec.AuditDir(), c.ScrapeBegin, c.ScrapeEnd)
				if err != nil {
					return err
				}

				if len(files) != 0 {
					delete(c.fileStats, "localhost")
					c.fileStats["localhost"] = make([]CollectStat, 0)
					auditLogSize := int64(0)
					auditLogCount := 0
					for _, v := range files {
						auditLogCount++
						auditLogSize = auditLogSize + v
					}

					c.fileStats["localhost"] = append(c.fileStats["localhost"], CollectStat{
						Target: fmt.Sprintf("%d TiUP %s audit logs", auditLogCount, c.topoType),
						Size:   auditLogSize,
					})
				}
				return nil
			},
		)

	t1 := task.NewBuilder(m.logger).
		ParallelStep(fmt.Sprintf("+ Collect TiUP %s audit log information", c.topoType), false,
			t.BuildAsStep(fmt.Sprintf("  - Scraping TiUP %s audit log", c.topoType))).
		Build()

	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)
	if err := t1.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return nil, err
		}
		return nil, perrs.Trace(err)
	}

	return c.fileStats, nil
}

// Collect implements the Collector interface
func (c *AuditLogCollectOptions) Collect(m *Manager, _ *models.TiDBCluster) error {
	if m.mode != CollectModeTiUP {
		return nil
	}

	t := task.NewBuilder(m.logger).
		Func(
			"collect audit log information",
			func(ctx context.Context) error {
				files, err := auditLogStat(spec.AuditDir(), c.ScrapeBegin, c.ScrapeEnd)
				if err != nil {
					return err
				}
				if len(files) != 0 {

					err := os.MkdirAll(filepath.Join(c.resultDir, fmt.Sprintf("%s_audit", c.topoType)), 0755)
					if err != nil {
						return err
					}

					for f := range files {
						utils.Copy(f, filepath.Join(c.resultDir, fmt.Sprintf("%s_audit", c.topoType), filepath.Base(f)))
					}
				}
				return nil
			},
		)

	t1 := task.NewBuilder(m.logger).
		ParallelStep("+ Scrap TiUP audit logs", false, t.BuildAsStep(fmt.Sprintf("  - copy TiUP %s audit log files", c.topoType))).
		Build()

	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)
	if err := t1.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

func auditLogStat(auditLogPath, scrapeBegin, scrapeEnd string) (scraper.FileStat, error) {
	var err error
	result := &scraper.Sample{}
	opt := &scraper.Option{
		FilePaths: []string{
			filepath.Join(auditLogPath, "*"),
		},
		Start: scrapeBegin,
		End:   scrapeEnd,
	}

	s := scraper.NewFileScraper(opt.FilePaths)
	if s.Start, err = utils.ParseTime(opt.Start); err != nil {
		return result.File, err
	}
	if s.End, err = utils.ParseTime(opt.End); err != nil {
		return result.File, err
	}

	s.Scrap(result)
	if err != nil {
		return result.File, err
	}
	return result.File, nil
}
