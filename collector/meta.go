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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	json "github.com/json-iterator/go"
	pingcapv1alpha1 "github.com/pingcap/diag/k8s/apis/pingcap/v1alpha1"
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/diag/version"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/api"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/meta"
)

const (
	FileNameClusterJSON       = "cluster.json"     // general cluster info
	FileNameTiUPClusterMeta   = "meta.yaml"        // tiup-cluster topology
	FileNameK8sClusterCRD     = "tidbcluster.json" // tidb-operator crd
	FileNameK8sClusterMonitor = "tidbmonitor.json" // tidb-operator crd
	DirNameSchema             = "db_vars"
)

// MetaCollectOptions is the options collecting cluster meta
type MetaCollectOptions struct {
	*BaseOptions
	rawRequest interface{}       // raw collect request or command
	opt        *operator.Options // global operations from cli
	session    string            // an unique session ID of the collection
	collectors map[string]bool
	resultDir  string
	tc         *pingcapv1alpha1.TidbCluster
	tm         *pingcapv1alpha1.TidbMonitor
}

type ClusterJSON struct {
	DiagVersion string              `json:"diag_version"`
	ClusterName string              `json:"cluster_name"`
	ClusterID   string              `json:"cluster_id"`   // the id from pd
	ClusterType string              `json:"cluster_type"` // tidb-cluster or dm-cluster
	DeployType  string              `json:"deploy_type"`  // deployment type
	Session     string              `json:"session"`
	BeginTime   string              `json:"begin_time"`
	EndTime     string              `json:"end_time"`
	Collectors  []string            `json:"collectors"`
	RawRequest  interface{}         `json:"raw_request"`
	Topology    *models.TiDBCluster `json:"topology"` // abstract cluster topo
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
func (c *MetaCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	// write cluster.json
	b := c.GetBaseOptions()
	var clusterID string
	var clusterType string
	var err error

	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)

	switch m.mode {
	case CollectModeTiUP:
		tiupTopo := topo.Attributes[CollectModeTiUP].(spec.Topology)
		clusterID, err = getTiUPClusterID(ctx, b.Cluster, m.tlsCfg)
		if err != nil {
			return err
		}
		clusterType = tiupTopo.Type()
	case CollectModeK8s:
		clusterID = c.tc.GetClusterID()
		clusterType = spec.TopoTypeTiDB
	default:
		// nothing
	}

	collectors := []string{}
	for name, enabled := range c.collectors {
		if enabled {
			collectors = append(collectors, name)
		}
	}

	jsonbyte, _ := json.MarshalIndent(ClusterJSON{
		DiagVersion: version.ShortVer(),
		ClusterName: b.Cluster,
		ClusterID:   clusterID,
		ClusterType: clusterType,
		DeployType:  m.mode,
		Session:     c.session,
		Collectors:  collectors,
		BeginTime:   b.ScrapeBegin,
		EndTime:     b.ScrapeEnd,
		RawRequest:  c.rawRequest,
		Topology:    topo,
	}, "", "  ")

	fn, err := os.Create(filepath.Join(c.resultDir, FileNameClusterJSON))
	if err != nil {
		return err
	}
	defer fn.Close()
	if _, err := fn.Write(jsonbyte); err != nil {
		return err
	}

	// save deployment related topology
	switch m.mode {
	case CollectModeTiUP:
		yamlMeta, err := os.ReadFile(m.specManager.Path(b.Cluster, "meta.yaml"))
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
		tcData, err := json.MarshalIndent(c.tc, "", "  ")
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
			tmData, err := json.MarshalIndent(c.tm, "", "  ")
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

func getTiUPClusterID(ctx context.Context, clusterName string, tlsCfg *tls.Config) (string, error) {
	metadata, err := spec.ClusterMetadata(clusterName)
	if err != nil && !errors.Is(perrs.Cause(err), meta.ErrValidate) &&
		!errors.Is(perrs.Cause(err), spec.ErrNoTiSparkMaster) {
		return "", err
	}

	pdEndpoints := make([]string, 0)
	for _, pd := range metadata.Topology.PDServers {
		pdEndpoints = append(pdEndpoints, fmt.Sprintf("%s:%d", pd.Host, pd.ClientPort))
	}

	pdAPI := api.NewPDClient(ctx, pdEndpoints, 2*time.Second, tlsCfg)
	id, err := pdAPI.GetClusterID()
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(id, 10), nil
}
