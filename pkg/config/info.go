// Copyright 2022 PingCAP, Inc.
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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/pingcap/diag/pkg/utils/toml"
)

// user specified diag config
type DiagInfo struct {
	ClinicServers map[Region]ClinicServer `toml:"clinicservers"`
}

type ClinicServer struct {
	Endpoint string `toml:"endpoint"`
	Cert     string `toml:"cert"`
	Info     string `toml:"info"`
}

var Info DiagInfo
var infoText string
var AvailableRegion []string

func init() {
	binpath, err := os.Executable()
	if err != nil {
		return
	}
	fp := filepath.Join(filepath.Dir(binpath), "info.toml")
	f, err := os.Open(fp)
	if err != nil {
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return
	}

	toml.Unmarshal(data, &Info)

	AvailableRegion, infoText = buildClinicServerInfo(Info.ClinicServers)
}

func buildClinicServerInfo(servers map[Region]ClinicServer) ([]string, string) {
	regions := make([]string, 0, len(servers))
	for region := range servers {
		regions = append(regions, string(region))
	}
	sort.Strings(regions)

	text := "Clinic Server provides the following regions to store your diagnostic data"
	for _, region := range regions {
		server := servers[Region(region)]
		text += fmt.Sprintf("\n[%s] region: %s url: %s", region, server.Info, server.Endpoint)
	}
	return regions, text
}

type Region string

func (r Region) Endpoint() string {
	return Info.ClinicServers[r].Endpoint
}

func (r Region) Cert() string {
	return Info.ClinicServers[r].Cert
}

func (r Region) Info() string {
	return Info.ClinicServers[r].Info
}
