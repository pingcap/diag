package alert

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type analyzeTask struct{}

func Analyze() *analyzeTask {
	return &analyzeTask{}
}

// Check if any table doest not have index
func (t *analyzeTask) Run(db *boot.DB, c *boot.Config) {
	rows, err := db.Query(
		`SELECT db, tb FROM inspection_db_info WHERE idx = 0 AND inspection = ?`,
		c.InspectionId,
	)
	if err != nil {
		log.Error("db.Query:", err)
		return
	}
	defer rows.Close()

	var dbn, table string
	for rows.Next() {
		if err := rows.Scan(&dbn, &table); err != nil {
			return
		}
		db.InsertSymptom(
			c.InspectionId,
			"error",
			fmt.Sprintf("table %s missing index in database %s", dbn, table),
			"please add index for the table",
		)
	}
}
