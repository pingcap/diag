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
	"time"

	"github.com/pingcap/tiup/pkg/base52"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	"github.com/pingcap/tiup/pkg/crypto/rand"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/tui"
)

// Manager to deploy a cluster.
type Manager struct {
	sysName     string
	specManager *spec.SpecManager
	session     string // an unique ID of the collection
	mode        string // tiup-cluster or tidb-operator
	diagMode    string // cmd or server
	logger      *logprinter.Logger
}

// NewManager create a Manager.
func NewManager(
	sysName string,
	specManager *spec.SpecManager,
	logger *logprinter.Logger,
) *Manager {
	currTime := time.Now()
	tid := base52.Encode(currTime.UnixNano() + rand.Int63n(1000))

	return &Manager{
		sysName:     sysName,
		specManager: specManager,
		session:     tid,
		logger:      logger,
	}
}

// NewEmptyManager creates a Manager with specific session ID and without initialing specManager
func NewEmptyManager(sysName, tid string, logger *logprinter.Logger) *Manager {
	return &Manager{
		sysName: sysName,
		session: tid,
		logger:  logger,
	}
}

func (m *Manager) sshTaskBuilder(name string, topo spec.Topology, user string, opts operator.Options) (*task.Builder, error) {
	var p *tui.SSHConnectionProps = &tui.SSHConnectionProps{}
	if opts.SSHType != executor.SSHTypeNone && len(opts.SSHProxyHost) != 0 {
		var err error
		if p, err = tui.ReadIdentityFileOrPassword(opts.SSHProxyIdentity, opts.SSHProxyUsePassword); err != nil {
			return nil, err
		}
	}

	return task.NewBuilder(m.logger).
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

const CollectLockName = ".collect.lock"

// collectLock when collecting data, add a file lock to mark that the collected data is incomplete
func (m *Manager) collectLock(resultDir string) {
	os.MkdirAll(resultDir, 0755)
	lockFile := filepath.Join(resultDir, CollectLockName)
	os.Create(lockFile)
}

// collectUnlock when the acquisition ends, remove the file lock
func (m *Manager) collectUnlock(resultDir string) {
	os.Remove(filepath.Join(resultDir, CollectLockName))
}
