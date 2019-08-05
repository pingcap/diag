package report

import (
	log "github.com/sirupsen/logrus"
)

type ConfigInfo struct {
	NodeIp    string `json:"node_ip"`
	Port      string `json:"port"`
	Component string `json:"component"`
	Config    string `json:"config"`
}

func (r *Report) loadConfigInfo() error {
	if !r.itemReady("basic") {
		return nil
	}

	rows, err := r.db.Query(
		`SELECT node_ip, port, component, config FROM software_config WHERE inspection = ?`,
		r.inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return err
	}

	infos := []*ConfigInfo{}
	for rows.Next() {
		info := ConfigInfo{}
		err = rows.Scan(&info.NodeIp, &info.Port, &info.Component, &info.Config)
		if err != nil {
			log.Error("db.Query:", err)
			return err
		}

		infos = append(infos, &info)
	}

	r.ConfigInfo = infos
	return nil
}
