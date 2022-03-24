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
	"github.com/pingcap/diag/pkg/models"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
)

// MetaDataCollectOptions are options used collecting component logs
type MetaDataCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	limit     int               // scp rate limit
	resultDir string
	fileStats map[string][]CollectStat
	compress  bool
}

// Desc implements the Collector interface
func (c *MetaDataCollectOptions) Desc() string {
	return "matedata of components"
}

// GetBaseOptions implements the Collector interface
func (c *MetaDataCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *MetaDataCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *MetaDataCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *MetaDataCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *MetaDataCollectOptions) Prepare(m *Manager, cls *models.TiDBCluster) (map[string][]CollectStat, error) {

	return c.fileStats, nil
}

// Collect implements the Collector interface
func (c *MetaDataCollectOptions) Collect(m *Manager, cls *models.TiDBCluster) error {

	return nil
}
