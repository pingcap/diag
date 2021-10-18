// Copyright 2021 PingCAP, Inc.
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

package collector

import (
	"fmt"

	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	"github.com/pingcap/tiup/pkg/set"
)

type hostInfo struct {
	ssh  int    // ssh port of host
	os   string // operating system
	arch string // cpu architecture
	// vendor string
}

// Deprecated
func convertStepDisplaysToTasks(t []*task.StepDisplay) []task.Task {
	tasks := make([]task.Task, 0, len(t))
	for _, sd := range t {
		tasks = append(tasks, sd)
	}
	return tasks
}

// buildDownloadCompTasks build download component tasks
func buildDownloadCompTasks(clusterVersion string, topo spec.Topology, bindVersion spec.BindVersion) []*task.StepDisplay {
	var tasks []*task.StepDisplay
	uniqueTaskList := set.NewStringSet()
	topo.IterInstance(func(inst spec.Instance) {
		key := fmt.Sprintf("%s-%s-%s", inst.ComponentName(), inst.OS(), inst.Arch())
		if found := uniqueTaskList.Exist(key); !found {
			uniqueTaskList.Insert(key)
			version := bindVersion(inst.ComponentName(), clusterVersion)

			t := task.NewBuilder().
				Download(inst.ComponentName(), inst.OS(), inst.Arch(), version).
				BuildAsStep(fmt.Sprintf("  - Download %s:%s (%s/%s)",
					inst.ComponentName(), version, inst.OS(), inst.Arch()))
			tasks = append(tasks, t)
		}
	})
	return tasks
}
