package alert

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type analyzeTask struct{}

func Analyze() *analyzeTask {
	return &analyzeTask{}
}

// Check if there is any alert from prometheus
func (t *analyzeTask) Run(db *boot.DB, c *boot.Config) {
	var count int
	if err := db.QueryRow(
		`SELECT count(*) FROM inspection_alerts WHERE inspection = ?`,
		c.InspectionId,
	).Scan(&count); err != nil {
		log.Error("db.QueryRow:", err)
		return
	}
	if count == 0 {
		return
	}
	msg := "there are alert information in the cluster"
	desc := "please check the alert information"
	db.InsertSymptom(c.InspectionId, "warning", msg, desc)
}
