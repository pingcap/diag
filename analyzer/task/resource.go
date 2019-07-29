package task

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/utils"
)

type Resource struct {
	AvgCPU float64
	MaxCPU float64
	AvgMem float64
	MaxMem float64
	AvgIoUtil float64
	MaxIoUtil float64
	AvgDisk float64
	MaxDisk float64
}

type ParseResourceTask struct {
	BaseTask
}

func ParseResource(base BaseTask) Task {
	return &ParseResourceTask {base}
}

func (t *ParseResourceTask) Run() error {
	cpu := t.ResourceUtil(
		fmt.Sprintf(`100 - avg (rate(node_cpu{mode="idle", inspectionid="%s"}[1m]) ) * 100`, t.inspectionId),
	)
	t.data.resource.AvgCPU = cpu.Avg()
	t.data.resource.MaxCPU = cpu.Max()

	mem := t.ResourceUtil(
		fmt.Sprintf(`100 - (sum(node_memory_MemAvailable{inspectionid="%s"}) / sum(node_memory_MemTotal{inspectionid="%s"})) * 100`, t.inspectionId),
	)
	t.data.resource.AvgMem = mem.Avg()
	t.data.resource.MaxMem = mem.Max()

	ioutil := t.ResourceUtil(
		fmt.Sprintf(`100 * (avg(max(rate(node_disk_io_time_ms{inspectionid="%s"}[1m]) / 1000) by (instance)))`, t.inspectionId),
	)
	t.data.resource.AvgIoUtil = ioutil.Avg()
	t.data.resource.MaxIoUtil = ioutil.Max()

	disk := t.ResourceUtil(
		fmt.Sprintf(
			`100 - (sum(node_filesystem_avail{fstype="ext4",inspectionid="%s"}) / sum(node_filesystem_size{fstype="ext4",inspectionid="%s"})) * 100`,
			t.inspectionId,
		),
	)
	t.data.resource.AvgDisk = disk.Avg()
	t.data.resource.MaxDisk = disk.Max()

	return nil
}

func (t *ParseResourceTask) ResourceUtil(query string) utils.FloatArray {
	inspectTime := time.Unix(int64(t.data.meta.InspectTime), 0)
	v, _ := utils.QueryPromRange(query, inspectTime.Add(-1 * time.Hour), inspectTime, time.Minute)
	return v
}