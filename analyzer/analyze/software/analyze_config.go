package software

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type analyzeConfigTask struct{}

func AnalyzeConfig() *analyzeConfigTask {
	return &analyzeConfigTask{}
}

// Check if all software configs are same
func (t *analyzeConfigTask) Run(db *boot.DB, c *boot.Config) {
	rows, err := db.Query(
		`SELECT component, COUNT(DISTINCT(config)) FROM software_config WHERE inspection = ? GROUP BY component`,
		c.InspectionId,
	)
	if err != nil {
		log.Error("db.QueryRow:", err)
		return
	}
	defer rows.Close()

	var name string
	var count int
	for rows.Next() {
		if err := rows.Scan(&name, &count); err != nil {
			return
		}
		if count == 1 {
			continue
		}
		msg := fmt.Sprintf("%s config is not identical", name)
		desc := fmt.Sprintf("make sure all %s have the same config", name)
		defer db.InsertSymptom(c.InspectionId, "error", msg, desc)
	}
}
