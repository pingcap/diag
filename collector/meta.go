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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	jsoniter "github.com/json-iterator/go"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/api"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/meta"
)

const (
	fileNameClusterMeta = "meta.yaml"
	fileNameClusterJSON = "cluster.json"
)

// MetaCollectOptions is the options collecting cluster meta
type MetaCollectOptions struct {
	*BaseOptions
	opt        *operator.Options // global operations from cli
	session    string            // an unique session ID of the collection
	collectors map[string]bool
	resultDir  string
	filePath   string
}

type ClusterJSON struct {
	ClusterName string   `json:"cluster_name"`
	ClusterID   int64    `json:"cluster_id"` // the id from pd
	Session     string   `json:"session"`
	BeginTime   string   `json:"begin_time"`
	EndTime     string   `json:"end_time"`
	Collectors  []string `json:"collectors"`
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
func (c *MetaCollectOptions) Prepare(_ *Manager, _ *spec.Specification) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *MetaCollectOptions) Collect(_ *Manager, _ *spec.Specification) error {
	// write cluster.json
	b := c.GetBaseOptions()
	clusterID, err := getClusterID(b.Cluster)
	if err != nil {
		fmt.Fprint(os.Stderr, fmt.Errorf("cannot get clusterID from PD"))
		return err
	}
	collectors := []string{}
	for name, enabled := range c.collectors {
		if enabled {
			collectors = append(collectors, name)
		}
	}

	jsonbyte, _ := jsoniter.MarshalIndent(ClusterJSON{
		ClusterName: b.Cluster,
		ClusterID:   clusterID,
		Session:     c.session,
		Collectors:  collectors,
		BeginTime:   b.ScrapeBegin,
		EndTime:     b.ScrapeEnd,
	}, "", "  ")

	fn, err := os.Create(filepath.Join(c.resultDir, fileNameClusterJSON))
	if err != nil {
		return err
	}
	defer fn.Close()
	if _, err := fn.Write(jsonbyte); err != nil {
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

func getClusterID(clusterName string) (int64, error) {
	metadata, err := spec.ClusterMetadata(clusterName)
	if err != nil && !errors.Is(perrs.Cause(err), meta.ErrValidate) &&
		!errors.Is(perrs.Cause(err), spec.ErrNoTiSparkMaster) {
		return 0, err
	}

	pdEndpoints := make([]string, 0)
	for _, pd := range metadata.Topology.PDServers {
		pdEndpoints = append(pdEndpoints, fmt.Sprintf("%s:%d", pd.Host, pd.ClientPort))
	}

	pdAPI := api.NewPDClient(pdEndpoints, 2*time.Second, nil)
	return pdAPI.GetClusterID()
}
