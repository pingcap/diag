package output

import (
	"github.com/pingcap/tidb-foresight/analyzer/output/alert"
	"github.com/pingcap/tidb-foresight/analyzer/output/basic"
	"github.com/pingcap/tidb-foresight/analyzer/output/dbinfo"
	"github.com/pingcap/tidb-foresight/analyzer/output/dmesg"
	"github.com/pingcap/tidb-foresight/analyzer/output/hardware"
	"github.com/pingcap/tidb-foresight/analyzer/output/inspection"
	"github.com/pingcap/tidb-foresight/analyzer/output/items"
	"github.com/pingcap/tidb-foresight/analyzer/output/logs"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	"github.com/pingcap/tidb-foresight/analyzer/output/network"
	"github.com/pingcap/tidb-foresight/analyzer/output/profile"
	"github.com/pingcap/tidb-foresight/analyzer/output/resource"
	"github.com/pingcap/tidb-foresight/analyzer/output/software"
)

func Tasks() []interface{} {
	tasks := make([]interface{}, 0)

	tasks = append(tasks, alert.SaveAlert())
	tasks = append(tasks, basic.SaveBasicInfo())
	tasks = append(tasks, dbinfo.SaveDBInfo())
	tasks = append(tasks, dmesg.SaveDmesg())
	tasks = append(tasks, hardware.SaveHardwareInfo())
	tasks = append(tasks, software.SaveSoftwareConfig())
	tasks = append(tasks, software.SaveSoftwareVersion())
	tasks = append(tasks, inspection.SaveInspection())
	tasks = append(tasks, resource.SaveResource())
	tasks = append(tasks, profile.SaveProfile())
	tasks = append(tasks, logs.CopyLogs())
	tasks = append(tasks, logs.SaveSlowQuery())
	tasks = append(tasks, network.SaveNetwork())
	tasks = append(tasks, items.SaveItems())
	tasks = append(tasks, metric.SaveMetric())

	return tasks
}
