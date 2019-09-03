package dbinfo

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/dbinfo"
	"github.com/pingcap/tidb-foresight/model"
	ti "github.com/pingcap/tidb-foresight/utils/tagd-value/int64"
	log "github.com/sirupsen/logrus"
)

type saveDBInfoTask struct{}

func SaveDBInfo() *saveDBInfoTask {
	return &saveDBInfoTask{}
}

// Save table indexes information to db, a record for a table
func (t *saveDBInfoTask) Run(m *boot.Model, schemas *dbinfo.DBInfo, c *boot.Config) {
	for _, schema := range *schemas {
		if schema.Name == "mysql" || schema.Name == "INFORMATION_SCHEMA" || schema.Name == "PERFORMANCE_SCHEMA" {
			continue
		}
		for _, tb := range schema.Tables {
			idxnum := ti.New(int64(len(tb.Indexes)), nil)
			if len(tb.Indexes) == 0 {
				idxnum.SetTag("status", "error")
				idxnum.SetTag("message", "please add index for this table")
			}
			if err := m.InsertInspectionDBInfo(&model.DBInfo{
				InspectionId: c.InspectionId,
				DB:           schema.Name,
				Table:        tb.Name.L,
				Index:        idxnum,
			}); err != nil {
				log.Error("insert db info:", err)
				return
			}
		}
	}
	return
}
