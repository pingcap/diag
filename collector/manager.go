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
	"time"

	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/base52"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	"github.com/pingcap/tiup/pkg/crypto/rand"
	"github.com/pingcap/tiup/pkg/tui"
)

// Manager to deploy a cluster.
type Manager struct {
	sysName     string
	specManager *spec.SpecManager
	session     string // an unique ID of the collection
	mode        string // tiup-cluster or tidb-operator
	DisplayMode string // display format
}

// NewManager create a Manager.
func NewManager(sysName string, specManager *spec.SpecManager) *Manager {
	currTime := time.Now()
	tid := base52.Encode(currTime.UnixNano() + rand.Int63n(1000))

	return &Manager{
		sysName:     sysName,
		specManager: specManager,
		session:     tid,
	}
}

// NewManager creates a Manager without initialing specManager
func NewEmptyManager(sysName string) *Manager {
	currTime := time.Now()
	tid := base52.Encode(currTime.UnixNano() + rand.Int63n(1000))

	return &Manager{
		sysName: sysName,
		session: tid,
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

func (m *Manager) sshTaskBuilder(name string, topo spec.Topology, user string, opts operator.Options) (*task.Builder, error) {
	var p *tui.SSHConnectionProps = &tui.SSHConnectionProps{}
	if opts.SSHType != executor.SSHTypeNone && len(opts.SSHProxyHost) != 0 {
		var err error
		if p, err = tui.ReadIdentityFileOrPassword(opts.SSHProxyIdentity, opts.SSHProxyUsePassword); err != nil {
			return nil, err
		}
	}

	return task.NewBuilder(m.DisplayMode).
		SSHKeySet(
			m.specManager.Path(name, "ssh", "id_rsa"),
			m.specManager.Path(name, "ssh", "id_rsa.pub"),
		).
		ClusterSSH(
			topo,
			user,
			opts.SSHTimeout,
			opts.OptTimeout,
			opts.SSHProxyHost,
			opts.SSHProxyPort,
			opts.SSHProxyUser,
			p.Password,
			p.IdentityFile,
			p.IdentityFilePassphrase,
			opts.SSHProxyTimeout,
			opts.SSHType,
			topo.BaseTopo().GlobalOptions.SSHType,
		), nil
}
