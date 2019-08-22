package resource

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/input/resource"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

const THRESHOLD = 20

type saveResourceTask struct {
}

// SaveResource returns an instance of saveResourceTask
func SaveResource() *saveResourceTask {
	return &saveResourceTask{}
}

// Insert resource usage to database for frontend presentation
func (t *saveResourceTask) Run(c *boot.Config, r *resource.Resource, args *args.Args, m *boot.Model) {
	d := utils.HumanizeDuration(args.ScrapeEnd.Sub(args.ScrapeBegin))
	if err := t.insertData(m, c.InspectionId, "cpu", d, r.AvgCPU); err != nil {
		log.Error("insert cpu usage:", err)
	}
	if err := t.insertData(m, c.InspectionId, "disk", d, r.AvgDisk); err != nil {
		log.Error("insert disk usage:", err)
	}
	if err := t.insertData(m, c.InspectionId, "ioutil", d, r.AvgIoUtil); err != nil {
		log.Error("insert ioutil usage:", err)
	}
	if err := t.insertData(m, c.InspectionId, "mem", d, r.AvgMem); err != nil {
		log.Error("insert memory usage:", err)
	}
}

func (t *saveResourceTask) insertData(m *boot.Model, inspectionId, resource, duration string, value float64) error {
	v := utils.NewTagdFloat64(value, nil)
	if v.GetValue() > THRESHOLD {
		v.SetTag("status", "abnormal")
		v.SetTag("message", fmt.Sprintf("%s Resource utilization/%s too high", resource, duration))
	}

	return m.InsertInspectionResourceInfo(&model.ResourceInfo{
		InspectionId: inspectionId,
		Name:         resource,
		Duration:     duration,
		Value:        v,
	})
}
