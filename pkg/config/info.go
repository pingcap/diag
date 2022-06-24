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
	"io"
	"os"
	"path/filepath"

	"github.com/pingcap/diag/pkg/utils/toml"
)

// user specified diag config
type DiagInfo struct {
	ClinicServers map[region]ClinicServer `toml:"clinicservers"`
}

type ClinicServer struct {
	Endpoint string `toml:"endpoint"`
	Cert     string `toml:"cert"`
	Info     string `toml:"info"`
}

var Info DiagInfo

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
}

/*
const RegionInfo = `Clinic Server provides the following two regions to store your diagnostic data:
[CN] region: Data stored in China Mainland, domain name : https://clinic.pingcap.com.cn
[US] region: Data stored in USA ,domain name : https://clinic.pingcap.com`

var RegionToEndpoint map[string]string = map[string]string{
	"CN": "https://clinic.pingcap.com.cn",
	"US": "https://clinic.pingcap.com",
}
*/
type region string

func (r region) Endpoint() string {
	return Info.ClinicServers[r].Endpoint
}

func (r region) Cert() string {
	return Info.ClinicServers[r].Cert
}

func (r region) Info() string {
	return Info.ClinicServers[r].Info
}
