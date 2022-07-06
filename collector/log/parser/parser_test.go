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
