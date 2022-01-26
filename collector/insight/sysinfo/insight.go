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

package sysinfo

import (
	"fmt"
	"runtime"
	"time"

	si "github.com/AstroProfundis/sysinfo"
	"github.com/pingcap/tidb-insight/collector/kmsg"
)

// Meta are information about insight itself
type Meta struct {
	Timestamp time.Time `json:"timestamp"`
	UPTime    float64   `json:"uptime,omitempty"`
	IdleTime  float64   `json:"idle_time,omitempty"`
	SiVer     string    `json:"sysinfo_ver"`
	GitBranch string    `json:"git_branch"`
	GitCommit string    `json:"git_commit"`
	GoVersion string    `json:"go_version"`
}

// InsightInfo are information gathered from the system
type InsightInfo struct {
	Meta       Meta          `json:"meta"`
	SysInfo    si.SysInfo    `json:"sysinfo,omitempty"`
	NTP        TimeStat      `json:"ntp,omitempty"`
	ChronyStat ChronyStat    `json:"chrony,omitempty"`
	Partitions []BlockDev    `json:"partitions,omitempty"`
	ProcStats  []ProcessStat `json:"proc_stats,omitempty"`
	EpollExcl  bool          `json:"epoll_exclusive,omitempty"`
	SysConfig  *SysCfg       `json:"system_configs,omitempty"`
	DMesg      []*kmsg.Msg   `json:"dmesg,omitempty"`
	Sockets    []Socket      `json:"sockets,omitempty"`
}

type Options struct {
	Pid    string
	Proc   bool
	Syscfg bool // collect kernel configs or not
	Dmesg  bool // collect kernel logs or not
}

func (info *InsightInfo) GetInfo(opts Options) {
	info.Meta.getMeta()
	info.SysInfo.GetSysInfo()
	info.NTP.getNTPInfo()
	info.ChronyStat.getChronyInfo()
	info.Partitions = GetPartitionStats()
	switch runtime.GOOS {
	case "android",
		"darwin",
		"dragonfly",
		"freebsd",
		"linux",
		"netbsd",
		"openbsd":
		info.EpollExcl = checkEpollExclusive()
	default:
		info.EpollExcl = false
	}

	// get process list
	if opts.Proc {
		info.ProcStats = GetProcessStats()
	}

	// get system configs
	if opts.Syscfg {
		info.SysConfig = &SysCfg{}
		info.SysConfig.getSysCfg()
	}

	// get kernel logs
	if opts.Dmesg {
		_ = info.collectDmsg()
	}

	_ = info.collectSockets()
}

func (meta *Meta) getMeta() {
	meta.Timestamp = time.Now()
	if sysUptime, sysIdleTime, err := GetSysUptime(); err == nil {
		meta.UPTime = sysUptime
		meta.IdleTime = sysIdleTime
	}

	meta.SiVer = si.Version
	meta.GitBranch = GitBranch
	meta.GitCommit = GitCommit
	meta.GoVersion = fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
