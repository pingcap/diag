package dbinfo

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/dbinfo"
	log "github.com/sirupsen/logrus"
)

type saveDBInfoTask struct{}

func SaveDBInfo() *saveDBInfoTask {
	return &saveDBInfoTask{}
}

// Save table indexes information to db, a record for a table
func (t *saveDBInfoTask) Run(db *boot.DB, schemas *dbinfo.DBInfo, c *boot.Config) {
	for _, schema := range *schemas {
		for _, tb := range schema.Tables {
			if _, err := db.Exec(
				"REPLACE INTO inspection_db_info(inspection, db, tb, idx) VALUES(?, ?, ?, ?)",
				c.InspectionId, schema.Name, tb.Name.L, len(tb.Indexes),
			); err != nil {
				log.Error("db.Exec:", err)
				return
			}
		}
	}
	return
}
