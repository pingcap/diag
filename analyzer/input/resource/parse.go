package resource

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	"github.com/pingcap/tidb-foresight/analyzer/utils"
)

type parseResourceTask struct{}

func ParseResource() *parseResourceTask {
	return &parseResourceTask{}
}

// Parse resource usage in the metric time range
func (t *parseResourceTask) Run(args *args.Args, c *boot.Config, m *metric.Metric /* DO NOT REMOVE THIS */) *Resource {
	resource := Resource{}

	cpu := t.resourceUtil(
		fmt.Sprintf(`100 - avg (rate(node_cpu{mode="idle", inspectionid="%s"}[1m]) ) * 100`, c.InspectionId),
		args.ScrapeBegin, args.ScrapeEnd,
	)
	resource.AvgCPU = cpu.Avg()
	resource.MaxCPU = cpu.Max()

	mem := t.resourceUtil(
		fmt.Sprintf(`100 - (sum(node_memory_MemAvailable{inspectionid="%s"}) / sum(node_memory_MemTotal{inspectionid="%s"})) * 100`, c.InspectionId),
		args.ScrapeBegin, args.ScrapeEnd,
	)
	resource.AvgMem = mem.Avg()
	resource.MaxMem = mem.Max()

	ioutil := t.resourceUtil(
		fmt.Sprintf(`100 * (avg(max(rate(node_disk_io_time_ms{inspectionid="%s"}[1m]) / 1000) by (instance)))`, c.InspectionId),
		args.ScrapeBegin, args.ScrapeEnd,
	)
	resource.AvgIoUtil = ioutil.Avg()
	resource.MaxIoUtil = ioutil.Max()

	disk := t.resourceUtil(
		fmt.Sprintf(
			`100 - (sum(node_filesystem_avail{fstype="ext4",inspectionid="%s"}) / sum(node_filesystem_size{fstype="ext4",inspectionid="%s"})) * 100`,
			c.InspectionId,
		),
		args.ScrapeBegin, args.ScrapeEnd,
	)
	resource.AvgDisk = disk.Avg()
	resource.MaxDisk = disk.Max()

	return &resource
}

func (t *parseResourceTask) resourceUtil(query string, begin, end time.Time) utils.FloatArray {
	v, _ := utils.QueryPromRange(query, begin, end, time.Minute)
	return v
}
