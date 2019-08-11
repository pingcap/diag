package clear

import (
	log "github.com/sirupsen/logrus"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
)

type clearHistoryTask struct{}

func ClearHistory() *clearHistoryTask {
	return &clearHistoryTask{}
}

// Delete records having the same inspection id for idempotency
func (t *clearHistoryTask) Run(c *boot.Config, db *boot.DB) {
	if _, err := db.Exec("DELETE FROM inspection_items WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete inspection_items:", err)
		return
	}

	if _, err := db.Exec("DELETE FROM inspection_symptoms WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete inspection_symptoms:", err)
		return
	}

	if _, err := db.Exec("DELETE FROM inspection_basic_info WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete inspection_basic_info:", err)
		return
	}

	if _, err := db.Exec("DELETE FROM inspection_db_info WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete inspection_db_info:", err)
		return
	}

	if _, err := db.Exec("DELETE FROM inspection_slow_log WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete inspection_slow_log:", err)
		return
	}

	if _, err := db.Exec("DELETE FROM inspection_network WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete inspection_network:", err)
		return
	}

	if _, err := db.Exec("DELETE FROM inspection_alerts WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete inspection_alerts:", err)
		return
	}

	if _, err := db.Exec("DELETE FROM inspection_hardware WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete inspection_hardware:", err)
		return
	}

	if _, err := db.Exec("DELETE FROM inspection_dmesg WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete inspection_dmesg:", err)
		return
	}

	if _, err := db.Exec("DELETE FROM software_version WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete software_version:", err)
		return
	}

	if _, err := db.Exec("DELETE FROM software_config WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete software_config:", err)
		return
	}

	if _, err := db.Exec("DELETE FROM inspection_resource WHERE inspection = ?", c.InspectionId); err != nil {
		log.Error("delete inspection_resource:", err)
		return
	}
}
