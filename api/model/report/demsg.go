package report

import (
	log "github.com/sirupsen/logrus"
)

type DemsgLog struct {
	NodeIp string `json:"node_ip"`
	Log    string `json:"log"`
}

func (r *Report) loadDemsgLog() error {
	if !r.itemReady("basic") {
		return nil
	}

	rows, err := r.db.Query(
		`SELECT node_ip, log FROM inspection_dmesg WHERE inspection = ?`,
		r.inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return err
	}

	logs := []*DemsgLog{}
	for rows.Next() {
		l := DemsgLog{}
		err = rows.Scan(&l.NodeIp, &l.Log)
		if err != nil {
			log.Error("db.Query:", err)
			return err
		}

		logs = append(logs, &l)
	}

	r.DemsgLog = logs
	return nil
}
