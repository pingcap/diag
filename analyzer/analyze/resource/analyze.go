package resource

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

const THRESHOLD = 20

type analyzeTask struct{}

func Analyze() *analyzeTask {
	return &analyzeTask{}
}

// Check if the avg resource useage exceed 20%
func (t *analyzeTask) Run(db *boot.DB, c *boot.Config) {
	rows, err := db.Query(
		`SELECT resource, duration FROM inspection_resource WHERE inspection = ? AND value > ?`,
		c.InspectionId, THRESHOLD,
	)
	if err != nil {
		log.Error("db.Query:", err)
		return
	}
	defer rows.Close()

	var resource, duration string
	for rows.Next() {
		if err := rows.Scan(&resource, &duration); err != nil {
			log.Error("db.Query:", err)
			return
		}
		msg := fmt.Sprintf("%s Resource utilization/%s exceeds %d%%",
			resource, duration, THRESHOLD)

		defer db.InsertSymptom(
			c.InspectionId,
			"error",
			msg,
			"please increase resources appropriately",
		)
	}
}
