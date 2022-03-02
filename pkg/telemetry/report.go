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

package telemetry

import (
	"runtime"

	"github.com/pingcap/diag/version"
)

// Report is the main telemetry message of diag
type Report struct {
	UUID          string       `json:"uuid,omitempty"`
	Version       *VersionInfo `json:"version,omitempty"`
	Command       string       `json:"command,omitempty"`
	CommandInfo   interface{}  `json:"command_info,omitempty"`
	ExecutionTime uint64       `json:"exec_time,omitempty"`
	ExitCode      int32        `json:"exit_code,omitempty"`
}

// VersionInfo holds the version of the particular binary
type VersionInfo struct {
	Release   string `json:"release,omitempty"`
	GitRef    string `json:"git_ref,omitempty"`
	GitCommit string `json:"git_commit,omitempty"`
	Os        string `json:"os,omitempty"`
	Arch      string `json:"arch,omitempty"`
	Go        string `json:"go,omitempty"`
}

// GetVersion populates VersionInfo
func GetVersion() *VersionInfo {
	return &VersionInfo{
		Release:   version.ReleaseVersion,
		GitRef:    version.GitBranch,
		GitCommit: version.GitHash,
		Os:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Go:        runtime.Version(),
	}
}

// CollectInfo is about the `collect` subcommand
type CollectInfo struct {
	ID         string   `json:"id,omitempty"` // hashed cluster name
	Mode       string   `json:"mode,omitempty"`
	ArgYes     bool     `json:"arg_yes,omitempty"`   // if the `--yes` argument is applied
	ArgLimit   int      `json:"arg_limit,omitempty"` // value of `-l/--limit` argument
	ArgInclude []string `json:"arg_include,omitempty"`
	ArgExclude []string `json:"arg_exclude,omitempty"`
	TimeSpan   int64    `json:"time_span,omitempty"`
	DataSize   int64    `json:"data_size,omitempty"` // data set size
	// TODO: add also estimated size before collecting
}

// PackageInfo is about the `package` subcommand
type PackageInfo struct {
	OriginalSize int64 `json:"original_size,omitempty"` // data set size
	PackageSize  int64 `json:"package_size,omitempty"`  // package size (compressed)
}

// UploadInfo is about the `upload` subcommand
type UploadInfo struct {
	PackageInfo

	Endpoint string `json:"endpoint,omitempty"`
}

// RebuildInfo is about the `rebuild` subcommand
type RebuildInfo struct {
	DataSize    int64 `json:"data_size,omitempty"`   // data set size
	Local       bool  `json:"local,omitempty"`       // if `--local` set
	Concurrency int   `json:"concurrency,omitempty"` // `-c`
}
