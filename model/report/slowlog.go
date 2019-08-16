package report

import (
	"time"

	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

type SlowLogInfo struct {
	Time  time.Time `json:"time"`
	Query string    `json:"query"`
}

func GetSlowLogInfo(db db.DB, inspectionId string) ([]*SlowLogInfo, error) {
	logs := []*SlowLogInfo{}

	rows, err := db.Query(
		`SELECT time, query, COUNT(time) as c FROM inspection_slow_log WHERE inspection = ? GROUP BY query ORDER BY c LIMIT 0, 20`,
		inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return logs, err
	}

	for rows.Next() {
		info := SlowLogInfo{}
		count := 0
		err = rows.Scan(&info.Time, &info.Query, &count)
		if err != nil {
			log.Error("db.Query:", err)
			return logs, err
		}

		logs = append(logs, &info)
	}

	return logs, nil
}
