package report

import (
	log "github.com/sirupsen/logrus"
)

type AlertInfo struct {
	Name       string `json:"name"`
	Value 		string `json:"value"`
	Time       string `json:"time"`
}

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
