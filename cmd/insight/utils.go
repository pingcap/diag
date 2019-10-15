package main

import (
	"io/ioutil"
	"strconv"
	"strings"
)

// Version infomation
var (
	// InsightGitBranch is initialized during make
	InsightGitBranch = "Not Provided"

	// InsightGitCommit is initialized during make
	InsightGitCommit = "Not Provided"

	// InsightBuildDate is initialized during make
	InsightBuildTime = "Not Provided"

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
	contents, err := ioutil.ReadFile(GetProcPath("uptime"))
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
