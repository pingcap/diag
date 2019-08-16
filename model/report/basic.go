package report

import (
	"database/sql"
	"strings"
	"time"

	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

type BasicInfo struct {
	ClusterName       string    `json:"cluster_name"`
	ClusterCreateTime time.Time `json:"cluster_create_time"`
	InspectTime       time.Time `json:"inspect_time"`
	TidbAlive         int       `json:"tidb_alive"`
	TikvAlive         int       `json:"tikv_alive"`
	PdAlive           int       `json:"pd_alive"`
	TidbCount         int       `json:"tidb_count"`
	TikvCount         int       `json:"tikv_count"`
	PdCount           int       `json:"pd_count"`
}

func GetBasicInfo(db db.DB, inspectionId string) (*BasicInfo, error) {
	basic := &BasicInfo{}

	if err := db.QueryRow(
		`SELECT cluster_name, cluster_create_t, inspect_t, tidb_count, tikv_count, pd_count FROM inspection_basic_info WHERE inspection = ?`,
		inspectionId,
	).Scan(&basic.ClusterName, &basic.ClusterCreateTime, &basic.InspectTime, &basic.TidbAlive, &basic.TikvAlive, &basic.PdAlive); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Error("db.QueryRow: ", err)
		return nil, err
	}

	var tidbs, tikvs, pds string
	if err := db.QueryRow(
		`SELECT tidb, tikv, pd FROM inspections WHERE id = ?`,
		inspectionId,
	).Scan(&tidbs, &tikvs, &pds); err != nil {
		log.Error("db.QueryRow: ", err)
		return nil, err
	}

	basic.TidbCount = len(strings.Split(tidbs, ","))
	basic.TikvCount = len(strings.Split(tikvs, ","))
	basic.PdCount = len(strings.Split(pds, ","))

	return basic, nil
}
