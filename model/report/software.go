package report

import (
	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

type SoftwareInfo struct {
	NodeIp    string `json:"node_ip"`
	Component string `json:"component"`
	Version   string `json:"version"`
}

// deprecated
func (r *Report) loadSoftwareInfo() error {
	if !r.itemReady("basic") {
		return nil
	}

	rows, err := r.db.Query(
		`SELECT node_ip, component, version FROM software_version WHERE inspection = ?`,
		r.inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return err
	}

	infos := []*SoftwareInfo{}
	for rows.Next() {
		info := SoftwareInfo{}
		err = rows.Scan(&info.NodeIp, &info.Component, &info.Version)
		if err != nil {
			log.Error("db.Query:", err)
			return err
		}

		infos = append(infos, &info)
	}

	r.SoftwareInfo = infos
	return nil
}

func GetSoftwareInfo(db db.DB, inspectionId string) ([]*SoftwareInfo, error) {
	infos := []*SoftwareInfo{}

	rows, err := db.Query(
		`SELECT node_ip, component, version FROM software_version WHERE inspection = ?`,
		inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return infos, err
	}

	for rows.Next() {
		info := SoftwareInfo{}
		err = rows.Scan(&info.NodeIp, &info.Component, &info.Version)
		if err != nil {
			log.Error("db.Query:", err)
			return infos, err
		}

		infos = append(infos, &info)
	}

	return infos, nil
}
