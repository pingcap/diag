package alert

import (
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/alert"
	log "github.com/sirupsen/logrus"
)

type saveAlertTask struct{}

func SaveAlert() *saveAlertTask {
	return &saveAlertTask{}
}

// Save alert information to database for future presentation
func (t *saveAlertTask) Run(c *boot.Config, alert *alert.Alert, db *boot.DB) {
	for _, alert := range *alert {
		if len(alert.Value) != 2 {
			continue
		}
		ts, ok := alert.Value[0].(float64)
		if !ok {
			log.Error("parse ts from alert failed")
			continue
		}
		v, ok := alert.Value[1].(string)
		if !ok {
			log.Error("parse value from alert failed")
			continue
		}
		if _, err := db.Exec(
			`INSERT INTO inspection_alerts(inspection, name, value, time) VALUES(?, ?, ?, ?)`,
			c.InspectionId, alert.Metric.Name, v, time.Unix(int64(ts), 0),
		); err != nil {
			log.Error("db.Exec:", err)
			return
		}
	}
}
