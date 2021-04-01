// Copyright 2020 PingCAP, Inc.
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
	"time"

	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/base52"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	"github.com/pingcap/tiup/pkg/utils/rand"
)

// Manager to deploy a cluster.
type Manager struct {
	sysName     string
	specManager *spec.SpecManager
	bindVersion spec.BindVersion
	session     string // an unique ID of the collection
}

// NewManager create a Manager.
func NewManager(sysName string, specManager *spec.SpecManager, bindVersion spec.BindVersion) *Manager {
	return &Manager{
		sysName:     sysName,
		specManager: specManager,
		bindVersion: bindVersion,
		session:     base52.Encode(time.Now().UnixNano() + rand.Int63n(1000)),
	}
}

func (m *Manager) meta(name string) (metadata spec.Metadata, err error) {
	exist, err := m.specManager.Exist(name)
	if err != nil {
		return nil, err
	}

	if !exist {
		return nil, perrs.Errorf("%s cluster `%s` not exists", m.sysName, name)
	}

	metadata = m.specManager.NewMetadata()
	err = m.specManager.Metadata(name, metadata)
	if err != nil {
		return metadata, err
	}

	return metadata, nil
}

func (m *Manager) sshTaskBuilder(name string, topo spec.Topology, user string, opts operator.Options) *task.Builder {
	return task.NewBuilder().
		SSHKeySet(
			m.specManager.Path(name, "ssh", "id_rsa"),
			m.specManager.Path(name, "ssh", "id_rsa.pub"),
		).
		ClusterSSH(topo, user, opts.SSHTimeout, opts.SSHType, topo.BaseTopo().GlobalOptions.SSHType)
}
