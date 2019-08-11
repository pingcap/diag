package analyze

import (
	"github.com/pingcap/tidb-foresight/analyzer/analyze/alert"
	"github.com/pingcap/tidb-foresight/analyzer/analyze/resource"
	"github.com/pingcap/tidb-foresight/analyzer/analyze/slow_query"
	"github.com/pingcap/tidb-foresight/analyzer/analyze/software"
	"github.com/pingcap/tidb-foresight/analyzer/analyze/summary"
	index "github.com/pingcap/tidb-foresight/analyzer/analyze/table_index"
)

func Tasks() []interface{} {
	tasks := make([]interface{}, 0)

	tasks = append(tasks, alert.Analyze())
	tasks = append(tasks, resource.Analyze())
	tasks = append(tasks, index.Analyze())
	tasks = append(tasks, software.AnalyzeConfig())
	tasks = append(tasks, software.AnalyzeVersion())
	tasks = append(tasks, slow_query.Analyze())

	// Keep this the last one
	tasks = append(tasks, summary.Summary())

	return tasks
}
