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
	"fmt"
	"os"

	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cliutil"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
)

// Collector is the configuration defining an collecting job
type Collector interface {
	Collect(*spec.Specification) error
	GetBaseOptions() *BaseOptions
	SetBaseOptions(*BaseOptions)
	Desc() string // a brief self description
}

// BaseOptions contains the options for check command
type BaseOptions struct {
	User        string                      // username to login to the SSH server
	UsePassword bool                        // use password instead of identity file for ssh connection
	SSH         *cliutil.SSHConnectionProps // SSH credentials
	ScrapeBegin string                      // start timepoint when collecting metrics and logs
	ScrapeEnd   string                      // stop timepoint when collecting metrics and logs
}

// CollectClusterInfo collects information and metrics from a tidb cluster
func (m *Manager) CollectClusterInfo(clusterName string, opt *BaseOptions, gOpt *operator.Options) error {
	var topo spec.Specification

	exist, err := m.specManager.Exist(clusterName)
	if err != nil {
		return err
	}
	if !exist {
		return perrs.Errorf("cluster %s does not exist", clusterName)
	}

	metadata, err := spec.ClusterMetadata(clusterName)
	if err != nil {
		return err
	}
	opt.User = metadata.User
	opt.SSH.IdentityFile = m.specManager.Path(clusterName, "ssh", "id_rsa")
	topo = *metadata.Topology

	// prepare output dir of collected data
	resultDir := m.specManager.Path(clusterName, "collector", m.session)
	if err := os.MkdirAll(resultDir, 0755); err != nil {
		return err
	}

	// build collector list
	collectors := make([]Collector, 0)

	// collect cluster topology
	collectors = append(collectors,
		&MetaCollectOptions{
			BaseOptions: opt,
			opt:         gOpt,
			resultDir:   resultDir,
			filePath:    m.specManager.Path(clusterName, "meta.yaml"),
		},
		&AlertCollectOptions{
			BaseOptions: opt,
			opt:         gOpt,
			resultDir:   resultDir,
		},
		&MetricCollectOptions{
			BaseOptions: opt,
			opt:         gOpt,
			resultDir:   resultDir,
		},
	)

	// collect data from remote servers
	var sshConnProps *cliutil.SSHConnectionProps = &cliutil.SSHConnectionProps{}
	if gOpt.SSHType != executor.SSHTypeNone {
		var err error
		if sshConnProps, err = cliutil.ReadIdentityFileOrPassword(opt.SSH.IdentityFile, opt.UsePassword); err != nil {
			return err
		}
	}
	opt.SSH = sshConnProps

	collectors = append(collectors, &SystemCollectOptions{
		BaseOptions: opt,
		opt:         gOpt,
		resultDir:   resultDir,
	})

	// run collectors
	for _, c := range collectors {
		fmt.Printf("Collecting %s...\n", c.Desc())
		if err := c.Collect(&topo); err != nil {
			return err
		}
	}

	fmt.Printf("Collected data are stored in %s\n", resultDir)
	return nil
}
