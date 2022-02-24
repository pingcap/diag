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
	"os"
	"strconv"
	"strings"
)

var (
	// Proc dir path for Linux
	procPath = "/proc"
)

func GetProcPath(paths ...string) string {
	switch len(paths) {
	case 0:
		return procPath
	default:
		all := make([]string, len(paths)+1)
		all[0] = procPath
		copy(all[1:], paths)
		return strings.Join(all, "/")
	}
}

func GetSysUptime() (float64, float64, error) {
	contents, err := os.ReadFile(GetProcPath("uptime"))
	if err != nil {
		return 0, 0, err
	}
	timerCounts := strings.Fields(string(contents))
	uptime, err := strconv.ParseFloat(timerCounts[0], 64)
	if err != nil {
		return 0, 0, err
	}
	idleTime, err := strconv.ParseFloat(timerCounts[1], 64)
	if err != nil {
		return 0, 0, err
	}
	return uptime, idleTime, err
}

// atoi converts a string to int, ignore any errors and return 0 on failure
func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}
