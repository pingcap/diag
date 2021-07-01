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
	"os"
	"path/filepath"

	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
)

const (
	fileNameClusterMeta = "meta.yaml"
	fileNameClusterName = "cluster-name.txt"
)

// MetaCollectOptions is the options collecting cluster meta
type MetaCollectOptions struct {
	*BaseOptions
	opt         *operator.Options // global operations from cli
	resultDir   string
	filePath    string
	clusterName string
}

// Desc implements the Collector interface
func (c *MetaCollectOptions) Desc() string {
	return "metadata of the cluster"
}

// GetBaseOptions implements the Collector interface
func (c *MetaCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *MetaCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *MetaCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *MetaCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *MetaCollectOptions) Prepare(topo *spec.Specification) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *MetaCollectOptions) Collect(topo *spec.Specification) error {
	// write cluster name to file
	fn, err := os.Create(filepath.Join(c.resultDir, fileNameClusterName))
	if err != nil {
		return err
	}
	defer fn.Close()
	if _, err := fn.Write([]byte(c.clusterName)); err != nil {
		return err
	}

	// save the topology
	yamlMeta, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}
	fm, err := os.Create(filepath.Join(c.resultDir, fileNameClusterMeta))
	if err != nil {
		return err
	}
	defer fm.Close()
	if _, err := fm.Write(yamlMeta); err != nil {
		return err
	}
	return nil
}
