package perf_map

import (
	"github.com/pingcap/tidb-foresight/analyzer/analyze/perf_map/tidb"
	"github.com/pingcap/tidb-foresight/analyzer/analyze/perf_map/tikv"
)

func Tasks() []interface{} {
	tasks := make([]interface{}, 0)

	tasks = append(tasks, tidb.Tasks()...)
	tasks = append(tasks, tikv.Tasks()...)

	return tasks
}
