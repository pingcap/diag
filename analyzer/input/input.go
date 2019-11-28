/**
 * Package will used by `analyzer/manager`, it parse the input files or arguments from file system
 * or analyzer/boot.
 * Then it runs
 */
package input

import (
	"github.com/pingcap/tidb-foresight/analyzer/input/alert"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/input/config"
	"github.com/pingcap/tidb-foresight/analyzer/input/dbinfo"
	"github.com/pingcap/tidb-foresight/analyzer/input/dmesg"
	"github.com/pingcap/tidb-foresight/analyzer/input/envs"
	"github.com/pingcap/tidb-foresight/analyzer/input/insight"
	"github.com/pingcap/tidb-foresight/analyzer/input/meta"
	"github.com/pingcap/tidb-foresight/analyzer/input/resource"
	"github.com/pingcap/tidb-foresight/analyzer/input/status"
	"github.com/pingcap/tidb-foresight/analyzer/input/topology"
)

func Tasks() []interface{} {
	tasks := make([]interface{}, 0)

	// parse input arguments, return `*Args`.
	tasks = append(tasks, args.ParseArgs())
	// read `DBInfo` from dbinfo directory.
	tasks = append(tasks, dbinfo.ParseDBInfo())
	// parse from topology.json.
	tasks = append(tasks, topology.ParseTopology())

	// parse from status.json
	tasks = append(tasks, status.ParseStatus())
	// parse from
	// ```
	// c *boot.Config, topo *model.Topology, args *args.Args, m *metric.Metric
	// ```
	// *and modify `topo *model.Topology`.*
	tasks = append(tasks, topology.ParseStatus())

	// read from prometheus to parse the resource
	tasks = append(tasks, resource.ParseResource())
	// parse from meta.json
	tasks = append(tasks, meta.ParseMeta())
	// parse from CountComponent
	tasks = append(tasks, meta.CountComponent())
	// parse from insight files.
	tasks = append(tasks, insight.ParseInsight())
	// parse from alert.json
	tasks = append(tasks, alert.ParseAlert())
	// parse from env.json
	tasks = append(tasks, envs.ParseEnvs())
	tasks = append(tasks, dmesg.ParseDmesg())
	tasks = append(tasks, config.ParseConfigInfo())

	return tasks
}
