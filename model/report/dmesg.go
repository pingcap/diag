package report

import (
	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

type DmesgLog struct {
	NodeIp string `json:"node_ip"`
	Log    string `json:"log"`
}

// deprecated
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

	logs := []*DmesgLog{}
	for rows.Next() {
		l := DmesgLog{}
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

func GetDmesgLog(db db.DB, inspectionId string) ([]*DmesgLog, error) {
	logs := []*DmesgLog{}

	rows, err := db.Query(
		`SELECT node_ip, log FROM inspection_dmesg WHERE inspection = ?`,
		inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return logs, err
	}

	for rows.Next() {
		l := DmesgLog{}
		err = rows.Scan(&l.NodeIp, &l.Log)
		if err != nil {
			log.Error("db.Query:", err)
			return logs, err
		}

		logs = append(logs, &l)
	}

	return logs, nil
}
