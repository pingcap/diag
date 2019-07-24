package report

import (
	log "github.com/sirupsen/logrus"
)

type BasicInfo struct {
	ClusterName       string `json:"cluster_name"`
	ClusterCreateTime string `json:"cluster_create_time"`
	InspectTime       string `json:"inspect_time"`
	TidbCount         int    `json:"tidb_count"`
	TikvCount         int    `json:"tikv_count"`
	PdCount           int    `json:"pd_count"`
}

func (r *Report) loadBasicInfo() error {
	if !r.itemReady("basic") {
		return nil
	}

	basic := &BasicInfo{}

	row := r.db.QueryRow(
		`SELECT cluster_name, cluster_create_t, inspect_t, tidb_count, tikv_count, pd_count FROM inspection_basic_info WHERE inspection = ?`,
		r.inspectionId,
	)

	err := row.Scan(&basic.ClusterName, &basic.ClusterCreateTime, &basic.InspectTime, &basic.TidbCount, &basic.TikvCount, &basic.PdCount)
	if err != nil {
		log.Error("db.QueryRow: ", err)
		return err
	}

	r.BasicInfo = basic
	return nil
}
