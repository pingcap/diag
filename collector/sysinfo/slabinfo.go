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

package sysinfo

// SlabInfo is an item of kernel slab allocator statistic, it is one
// line as in /proc/slabinfo
// We only support slabinfo v2.1, which first appeared in Linux 2.6.10
// and has not changed until Linux 5.16.
// For detailed documentation, refer to slabinfo(5) manual.
type SlabInfo struct {
	Name         string        `json:"name"`
	ActiveObjs   int           `json:"active_objs"`
	NumObjs      int           `json:"num_objs"`
	ObjSize      int           `json:"objsize"`
	ObjPerSlab   int           `json:"objperslab"`
	PagesPerSlab int           `json:"pagesperslab"`
	Tunables     TunableParams `json:"tunables"`
	SlabData     SlabData      `json:"slabdata"`
}

// TunableParams is the tunable parameters for the corresponding cache
type TunableParams struct {
	Limit        int `json:"limit"`
	BatchCount   int `json:"batchcount"`
	SharedFactor int `json:"sharedfactor"`
}

// SlabData contain detailed data of the curresponding cache
type SlabData struct {
	ActiveSlabs int `json:"active_slabs"`
	NumSlabs    int `json:"nums_slab"`
	SharedAvail int `json:"sharedavail"`
}

func (info *InsightInfo) getSlabInfo() error {
	slabInfo, err := GetSlabInfo()
	if err != nil {
		return err
	}
	info.SlabInfo = slabInfo
	return nil
}
