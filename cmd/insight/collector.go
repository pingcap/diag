// Copyright 2018 PingCAP, Inc.
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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/jaypipes/ghw"
	"github.com/pingcap/tidb-foresight/cmd/insight/osconfig"
	"github.com/pingcap/tidb-foresight/cmd/insight/sysinfo"
)

type Meta struct {
	Timestamp time.Time  `json:"timestamp"`
	UPTime    float64    `json:"uptime,omitempty"`
	IdleTime  float64    `json:"idle_time,omitempty"`
	SiVer     string     `json:"sysinfo_ver"`
	GitBranch string     `json:"git_branch"`
	GitCommit string     `json:"git_commit"`
	BuildTime string     `json:"utc_build_time"`
	GoVersion string     `json:"go_version"`
	TiDBVer   []TiDBMeta `json:"tidb"`
	TiKVVer   []TiKVMeta `json:"tikv"`
	PDVer     []PDMeta   `json:"pd"`
}

type Metrics struct {
	Meta       Meta            `json:"meta"`
	SysInfo    sysinfo.SysInfo `json:"sysinfo,omitempty"`
	NTP        TimeStat        `json:"ntp,omitempty"`
	Partitions []BlockDev      `json:"partitions,omitempty"`
	ProcStats  []ProcessStat   `json:"proc_stats,omitempty"`

	BlockInfo  ghw.BlockInfo `json:"block_info"`
	OsConfig osconfig.OsConfig `json:"os_config"`
}

type options struct {
	Port string
	Proc bool
}

func parseOpts() options {
	optPort := flag.String("port", "", "The port of process to collect info. Multiple ports can be separated by ','.")
	optProc := flag.Bool("proc", false, "Only collect process info, disabled (Collect everything except process info) by default.")
	flag.Parse()

	opts := options{*optPort, *optProc}
	return opts
}

func main() {
	opts := parseOpts()

	var metric Metrics
	metric.getMetrics(opts)

	data, err := json.MarshalIndent(&metric, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
}

func (metric *Metrics) getMetrics(opts options) {
	var portList []string
	if len(opts.Port) > 0 {
		portList = strings.Split(opts.Port, ",")
	}

	metric.Meta.getMeta(portList)
	if opts.Proc {
		metric.ProcStats = GetProcessStats(portList)
	} else {
		metric.SysInfo.GetSysInfo()
		metric.NTP.getNTPInfo()
		metric.Partitions = GetPartitionStats()
		metric.OsConfig = osconfig.Collect()
		info, err := ghw.Block()
		if err == nil {
			metric.BlockInfo = *info
		}
	}
}

func (meta *Meta) getMeta(portList []string) {
	meta.Timestamp = time.Now()
	if sysUptime, sysIdleTime, err := GetSysUptime(); err == nil {
		meta.UPTime = sysUptime
		meta.IdleTime = sysIdleTime
	}

	meta.SiVer = sysinfo.Version
	meta.GitBranch = InsightGitBranch
	meta.GitCommit = InsightGitCommit
	meta.BuildTime = InsightBuildTime
	meta.GoVersion = fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if len(portList) > 0 {
		meta.TiDBVer = getTiDBVersionByPortList(portList)
		meta.TiKVVer = getTiKVVersionByPortList(portList)
		meta.PDVer = getPDVersionByPortList(portList)
	} else {
		meta.TiDBVer = getTiDBVersionByName()
		meta.TiKVVer = getTiKVVersionByName()
		meta.PDVer = getPDVersionByName()
	}
}
