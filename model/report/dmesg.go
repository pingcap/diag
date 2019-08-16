package report

import (
	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

type DmesgLog struct {
	NodeIp string `json:"node_ip"`
	Log    string `json:"log"`
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
