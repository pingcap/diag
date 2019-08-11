package resource

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/input/resource"
	"github.com/pingcap/tidb-foresight/analyzer/utils"
	log "github.com/sirupsen/logrus"
)

type saveResourceTask struct {
}

// SaveResource returns an instance of saveResourceTask
func SaveResource() *saveResourceTask {
	return &saveResourceTask{}
}

// Insert resource usage to database for frontend presentation
func (t *saveResourceTask) Run(c *boot.Config, r *resource.Resource, args *args.Args, db *boot.DB) {
	d := utils.HumanizeDuration(args.ScrapeEnd.Sub(args.ScrapeBegin))
	if err := t.insertData(db, c.InspectionId, "cpu", d, r.AvgCPU); err != nil {
		log.Error("insert cpu usage:", err)
		return
	}
	if err := t.insertData(db, c.InspectionId, "disk", d, r.AvgDisk); err != nil {
		log.Error("insert disk usage:", err)
		return
	}
	if err := t.insertData(db, c.InspectionId, "ioutil", d, r.AvgIoUtil); err != nil {
		log.Error("insert ioutil usage:", err)
		return
	}
	if err := t.insertData(db, c.InspectionId, "mem", d, r.AvgMem); err != nil {
		log.Error("insert memory usage:", err)
		return
	}
}

func (t *saveResourceTask) insertData(db *boot.DB, inspectionId, resource, duration string, value float64) error {
	if _, err := db.Exec(
		"INSERT INTO inspection_resource(inspection, resource, duration, value) VALUES(?, ?, ?, ?)",
		inspectionId, resource, duration, value,
	); err != nil {
		log.Error("db:Exec:", err)
		return err
	}
	return nil
}
