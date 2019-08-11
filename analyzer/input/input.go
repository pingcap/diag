package input

import (
	"github.com/pingcap/tidb-foresight/analyzer/input/alert"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/input/dbinfo"
	"github.com/pingcap/tidb-foresight/analyzer/input/envs"
	"github.com/pingcap/tidb-foresight/analyzer/input/insight"
	"github.com/pingcap/tidb-foresight/analyzer/input/meta"
	"github.com/pingcap/tidb-foresight/analyzer/input/resource"
	"github.com/pingcap/tidb-foresight/analyzer/input/status"
	"github.com/pingcap/tidb-foresight/analyzer/input/topology"
)

func Tasks() []interface{} {
	tasks := make([]interface{}, 0)

	tasks = append(tasks, args.ParseArgs())
	tasks = append(tasks, dbinfo.ParseDBInfo())
	tasks = append(tasks, topology.ParseTopology())
	tasks = append(tasks, status.ParseStatus())
	tasks = append(tasks, resource.ParseResource())
	tasks = append(tasks, meta.ParseMeta())
	tasks = append(tasks, insight.ParseInsight())
	tasks = append(tasks, alert.ParseAlert())
	tasks = append(tasks, envs.ParseEnvs())

	return tasks
}
