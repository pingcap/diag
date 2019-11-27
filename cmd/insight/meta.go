package main

import (
	"github.com/shirou/gopsutil/process"
)


// This struct will be used as base of `TiKVMeta` `TiDBMeta` and `PDMeta`
type MetaBase struct {
	Pid        int32  `json:"pid,omitempty"`
	ReleaseVer string `json:"release_version,omitempty"`
	GitCommit  string `json:"git_commit,omitempty"`
	GitBranch  string `json:"git_branch,omitempty"`
	BuildTime  string `json:"utc_build_time,omitempty"`

	OpenFile uint64 `json:"open_file"`
	OpenFileLimit int32 `json:"open_file_limit"`
}

func (mb *MetaBase) ParseLimits(p *process.Process) error  {
	rlimits, err := p.Rlimit()
	if err != nil {
		return err
	}
	fsize := rlimits[process.RLIMIT_FSIZE]

	mb.OpenFile = fsize.Used
	mb.OpenFileLimit = fsize.Soft

	return nil
}