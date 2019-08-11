package software

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type analyzeVersionTask struct{}

func AnalyzeVersion() *analyzeVersionTask {
	return &analyzeVersionTask{}
}

// Check if all software versions are same
func (t *analyzeVersionTask) Run(db *boot.DB, c *boot.Config) {
	rows, err := db.Query(
		`SELECT component, COUNT(DISTINCT(version)) FROM software_version WHERE inspection = ? GROUP BY component`,
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
		msg := fmt.Sprintf("%s versions are not identical", name)
		desc := fmt.Sprintf("make sure all %s have the same version", name)
		defer db.InsertSymptom(c.InspectionId, "error", msg, desc)
	}
}
