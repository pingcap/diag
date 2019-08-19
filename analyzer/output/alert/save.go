package alert

import (
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/alert"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type saveAlertTask struct{}

func SaveAlert() *saveAlertTask {
	return &saveAlertTask{}
}

// Save alert information to database for future presentation
func (t *saveAlertTask) Run(c *boot.Config, alert *alert.Alert, m *boot.Model) {
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
		info := &model.AlertInfo{
			InspectionId: c.InspectionId,
			Name:         alert.Metric.Name,
			Value:        v,
			Time:         time.Unix(int64(ts), 0),
		}
		if err := m.InsertInspectionAlertInfo(info); err != nil {
			log.Error("insert alert info:", err)
			return
		}
	}
}
