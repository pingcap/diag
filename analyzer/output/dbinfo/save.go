package dbinfo

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/dbinfo"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type saveDBInfoTask struct{}

func SaveDBInfo() *saveDBInfoTask {
	return &saveDBInfoTask{}
}

// Save table indexes information to db, a record for a table
func (t *saveDBInfoTask) Run(m *boot.Model, schemas *dbinfo.DBInfo, c *boot.Config) {
	for _, schema := range *schemas {
		for _, tb := range schema.Tables {
			if err := m.InsertInspectionDBInfo(&model.DBInfo{
				InspectionId: c.InspectionId,
				DB:           schema.Name,
				Table:        tb.Name.L,
				Index:        len(tb.Indexes),
			}); err != nil {
				log.Error("insert db info:", err)
				return
			}
		}
	}
	return
}
