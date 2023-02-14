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

//go:build linux
// +build linux

package sysinfo

import (
	"os"
	"strings"
)

const slabInfoFilePath = "slabinfo"

// GetSlabInfo reads /proc/cpuinfo and parses its content
func GetSlabInfo() ([]SlabInfo, error) {
	content, err := os.ReadFile(GetProcPath(slabInfoFilePath))
	if err != nil {
		return nil, err
	}

	result := make([]SlabInfo, 0)
	for i, line := range strings.Split(string(content), "\n") {
		if i <= 1 { // the first 2 lines are headers
			continue
		}

		slabInfo := SlabInfo{}
		// name <active_objs> <num_objs> <objsize> <objperslab> \
		// <pagesperslab> : tunables <limit> <batchcount> <sharedfactor> : \
		// slabdata <active_slabs> <num_slabs> <sharedavail>
		fields := strings.Fields(line)
		if len(fields) < 16 {
			continue
		}

		slabInfo.Name = fields[0]
		slabInfo.ActiveObjs = atoi(fields[1])
		slabInfo.NumObjs = atoi(fields[2])
		slabInfo.ObjSize = atoi(fields[3])
		slabInfo.ObjPerSlab = atoi(fields[4])
		slabInfo.PagesPerSlab = atoi(fields[5])
		slabInfo.Tunables = TunableParams{
			Limit:        atoi(fields[8]),
			BatchCount:   atoi(fields[9]),
			SharedFactor: atoi(fields[10]),
		}
		slabInfo.SlabData = SlabData{
			ActiveSlabs: atoi(fields[13]),
			NumSlabs:    atoi(fields[14]),
			SharedAvail: atoi(fields[15]),
		}

		result = append(result, slabInfo)
	}
	return result, nil
}
