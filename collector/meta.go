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
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"
	pingcapv1alpha1 "github.com/pingcap/diag/k8s/apis/pingcap/v1alpha1"
	"github.com/pingcap/diag/pkg/models"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/api"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/meta"
)

const (
	FileNameClusterAbstractTopo = "topology.json"    // abstract topology
	FileNameClusterJSON         = "cluster.json"     // general cluster info
	FileNameTiUPClusterMeta     = "meta.yaml"        // tiup-cluster topology
	FileNameK8sClusterCRD       = "tidbcluster.json" // tidb-operator crd
	FileNameK8sClusterMonitor   = "tidbmonitor.json" // tidb-operator crd
	DirNameInfoSchema           = "info_schema"
)

// MetaCollectOptions is the options collecting cluster meta
type MetaCollectOptions struct {
	*BaseOptions
	opt        *operator.Options // global operations from cli
	session    string            // an unique session ID of the collection
	collectors map[string]bool
	resultDir  string
	filePath   string
	cluster    *models.TiDBCluster
	tc         *pingcapv1alpha1.TidbCluster
	tm         *pingcapv1alpha1.TidbMonitor
}

type ClusterJSON struct {
	ClusterName string   `json:"cluster_name"`
	ClusterID   int64    `json:"cluster_id"`  // the id from pd
	DeployType  string   `json:"deploy_type"` // deployment type
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
func (c *MetaCollectOptions) Prepare(_ *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *MetaCollectOptions) Collect(m *Manager, _ *models.TiDBCluster) error {
	// write cluster.json
	b := c.GetBaseOptions()
	var clusterID int64
	var err error

	switch m.mode {
	case CollectModeTiUP:
		clusterID, err = getTiUPClusterID(b.Cluster)
	case CollectModeK8s:
		var id int
		id, err = strconv.Atoi(c.tc.GetClusterID())
		clusterID = int64(id)
	default:
		// nothing
	}
	if err != nil {
		fmt.Fprint(os.Stderr, fmt.Errorf("cannot get clusterID from PD"))
		return err
	}
	c.cluster.ID = clusterID

	collectors := []string{}
	for name, enabled := range c.collectors {
		if enabled {
			collectors = append(collectors, name)
		}
	}

	jsonbyte, _ := jsoniter.MarshalIndent(ClusterJSON{
		ClusterName: b.Cluster,
		ClusterID:   clusterID,
		DeployType:  m.mode,
		Session:     c.session,
		Collectors:  collectors,
		BeginTime:   b.ScrapeBegin,
		EndTime:     b.ScrapeEnd,
	}, "", "  ")

	fn, err := os.Create(filepath.Join(c.resultDir, FileNameClusterJSON))
	if err != nil {
		return err
	}
	defer fn.Close()
	if _, err := fn.Write(jsonbyte); err != nil {
		return err
	}

	// save the topology
	// save the abstract topology
	clsData, err := jsoniter.MarshalIndent(c.cluster, "", "  ")
	if err != nil {
		return err
	}
	fcls, err := os.Create(filepath.Join(c.resultDir, FileNameClusterAbstractTopo))
	if err != nil {
		return err
	}
	defer fcls.Close()
	if _, err := fcls.Write(clsData); err != nil {
		return err
	}

	// save deployment related topology
	switch m.mode {
	case CollectModeTiUP:
		yamlMeta, err := os.ReadFile(c.filePath)
		if err != nil {
			return err
		}
		fm, err := os.Create(filepath.Join(c.resultDir, FileNameTiUPClusterMeta))
		if err != nil {
			return err
		}
		defer fm.Close()
		if _, err := fm.Write(yamlMeta); err != nil {
			return err
		}
	case CollectModeK8s:
		tcData, err := jsoniter.MarshalIndent(c.tc, "", "  ")
		if err != nil {
			return err
		}
		fc, err := os.Create(filepath.Join(c.resultDir, FileNameK8sClusterCRD))
		if err != nil {
			return err
		}
		defer fc.Close()
		if _, err := fc.Write(tcData); err != nil {
			return err
		}
		if c.tm != nil {
			tmData, err := jsoniter.MarshalIndent(c.tm, "", "  ")
			if err != nil {
				return err
			}
			fm, err := os.Create(filepath.Join(c.resultDir, FileNameK8sClusterMonitor))
			if err != nil {
				return err
			}
			defer fm.Close()
			if _, err := fm.Write(tmData); err != nil {
				return err
			}
		}
	}
	return nil
}

func getTiUPClusterID(clusterName string) (int64, error) {
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
