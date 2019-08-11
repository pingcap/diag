package summary

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type summaryTask struct{}

func Summary() *summaryTask {
	return &summaryTask{}
}

// Change the inspection status to success
func (t *summaryTask) Run(db *boot.DB, c *boot.Config) {
	if _, err := db.Exec(
		`UPDATE inspection_items SET status = 'success' WHERE inspection = ? AND status = 'running'`,
		c.InspectionId,
	); err != nil {
		log.Error("db.Exec:", err)
		return
	}

	if _, err := db.Exec(
		"UPDATE inspections SET status = ? WHERE id = ?",
		"success", c.InspectionId,
	); err != nil {
		log.Panic("update inspection status:", err)
	}
}
