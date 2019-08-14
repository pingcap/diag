package report

import (
	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

type AlertInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Time  string `json:"time"`
}

// deprecated
func (r *Report) loadAlertInfo() error {
	if !r.itemReady("metric") {
		return nil
	}

	rows, err := r.db.Query(
		`SELECT name, value, time FROM inspection_alerts WHERE inspection = ?`,
		r.inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return err
	}

	alerts := []*AlertInfo{}
	for rows.Next() {
		alert := AlertInfo{}
		err = rows.Scan(&alert.Name, &alert.Value, &alert.Time)
		if err != nil {
			log.Error("db.Query:", err)
			return err
		}

		alerts = append(alerts, &alert)
	}

	r.AlertInfo = alerts

	return nil
}

func GetAlertInfo(db db.DB, inspectionId string) ([]*AlertInfo, error) {
	alerts := []*AlertInfo{}

	rows, err := db.Query(
		`SELECT name, value, time FROM inspection_alerts WHERE inspection = ?`,
		inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return alerts, err
	}

	for rows.Next() {
		alert := AlertInfo{}
		err = rows.Scan(&alert.Name, &alert.Value, &alert.Time)
		if err != nil {
			log.Error("db.Query:", err)
			return alerts, err
		}

		alerts = append(alerts, &alert)
	}

	return alerts, nil
}
