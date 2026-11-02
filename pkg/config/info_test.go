// Copyright 2026 PingCAP, Inc.
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

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildClinicServerInfo(t *testing.T) {
	servers := map[Region]ClinicServer{
		"US": {
			Endpoint: "https://clinic.pingcap.com",
			Info:     "Data stored in USA",
		},
		"CN2": {
			Endpoint: "https://clinic.pingkai.cn",
			Info:     "Data stored in China Mainland",
		},
		"CN": {
			Endpoint: "https://clinic.pingcap.com.cn",
			Info:     "Data stored in China Mainland",
		},
	}

	regions, text := buildClinicServerInfo(servers)

	require.Equal(t, []string{"CN", "CN2", "US"}, regions)
	require.Equal(t, `Clinic Server provides the following regions to store your diagnostic data
[CN] region: Data stored in China Mainland url: https://clinic.pingcap.com.cn
[CN2] region: Data stored in China Mainland url: https://clinic.pingkai.cn
[US] region: Data stored in USA url: https://clinic.pingcap.com`, text)
}
