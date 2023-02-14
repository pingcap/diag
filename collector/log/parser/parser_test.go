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

package parser

import (
	"testing"
	"time"

	"github.com/pingcap/diag/collector/log/item"
	"github.com/stretchr/testify/require"
)

func TestPrometheusLogParser(t *testing.T) {
	assert := require.New(t)

	head := []byte(`level=warn ts=2022-06-09T09:16:54.674Z caller=main.go:377 deprecation_notice="'storage.tsdb.retention' flag is deprecated use 'storage.tsdb.retention.time' instead."`)
	p := &PrometheusLogParser{}
	ht, level := p.ParseHead(head)
	assert.Equal(item.LevelWARN, level)
	assert.Equal("2022-06-09T09:16:54.674Z", ht.Format(time.RFC3339Nano))
}

func TestPDV2LogParser(t *testing.T) {
	assert := require.New(t)

	head := []byte(`2020/11/27 12:00:31.898 main.go:101: [fatal] run server failed: listen tcp 127.0.0.1:2380: bind: address already in use`)
	p := &PDLogV2Parser{}
	ht, level := p.ParseHead(head)
	assert.Equal(item.LevelFATAL, level)
	assert.Equal("2020-11-27T12:00:31.898", ht.Format("2006-01-02T15:04:05.999999999")) // RFC3339Nano wo/ tz
}

func TestUnifiedLogParser(t *testing.T) {
	assert := require.New(t)

	p := &UnifiedLogParser{}

	// PD
	head := []byte(`[2022/12/04 21:37:15.834 +08:00] [WARN] [config_logging.go:287] ["rejected connection"] [remote-addr=172.16.4.128:43316] [server-name=] [error="tls: first record does not look like a TLS handshake"]`)
	ht, level := p.ParseHead(head)
	assert.Equal(item.LevelWARN, level)
	assert.Equal("2022-12-04T21:37:15.834+08:00", ht.Format(time.RFC3339Nano))

	// TiKV
	head = []byte(`[2022/08/17 17:10:13.258 +08:00] [INFO] [mod.rs:74] ["cgroup quota: memory=Some(9223372036854771712), cpu=None, cores={2, 1, 3, 0}"]`)
	ht, level = p.ParseHead(head)
	assert.Equal(item.LevelINFO, level)
	assert.Equal("2022-08-17T17:10:13.258+08:00", ht.Format(time.RFC3339Nano))

	// TiDB
	head = []byte(`[2022/12/06 13:49:28.842 +08:00] [FATAL] [bootstrap.go:1096] ["doReentrantDDL error"] [error="[ddl:8204]invalid ddl job type: none"] [stack="github.com/pingcap/tidb/session.doReentrantDDL\n\t/home/jenkins/agent/workspace/build-common/go/src/github.com/pingcap/tidb/session/bootstrap.go:1096\ngithub.com/pingcap/tidb/session.upgradeToVer107\n\t/home/jenkins/agent/workspace/build-common/go/src/github.com/pingcap/tidb/session/bootstrap.go:2181\ngithub.com/pingcap/tidb/session.upgrade\n\t/home/jenkins/agent/workspace/build-common/go/src/github.com/pingcap/tidb/session/bootstrap.go:941\ngithub.com/pingcap/tidb/session.runInBootstrapSession\n\t/home/jenkins/agent/workspace/build-common/go/src/github.com/pingcap/tidb/session/session.go:3114\ngithub.com/pingcap/tidb/session.BootstrapSession\n\t/home/jenkins/agent/workspace/build-common/go/src/github.com/pingcap/tidb/session/session.go:2966\nmain.createStoreAndDomain\n\t/home/jenkins/agent/workspace/build-common/go/src/github.com/pingcap/tidb/tidb-server/main.go:312\nmain.main\n\t/home/jenkins/agent/workspace/build-common/go/src/github.com/pingcap/tidb/tidb-server/main.go:213\nruntime.main\n\t/usr/local/go/src/runtime/proc.go:250"]`)
	ht, level = p.ParseHead(head)
	assert.Equal(item.LevelFATAL, level)
	assert.Equal("2022-12-06T13:49:28.842+08:00", ht.Format(time.RFC3339Nano))
}
