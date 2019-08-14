package report

import (
	log "github.com/sirupsen/logrus"
)

type HardwareInfo struct {
	NodeIp  string `json:"node_ip"`
	Cpu     string `json:"cpu"`
	Memory  string `json:"memory"`
	Disk    string `json:"disk"`
	Network string `json:"network"`
}

func (r *Report) loadHardwareInfo() error {
	if !r.itemReady("basic") {
		return nil
	}

	rows, err := r.db.Query(
		`SELECT node_ip, cpu, memory, disk, network FROM inspection_hardware WHERE inspection = ?`,
		r.inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return err
	}

	infos := []*HardwareInfo{}
	for rows.Next() {
		info := HardwareInfo{}
		err = rows.Scan(&info.NodeIp, &info.Cpu, &info.Memory, &info.Disk, &info.Network)
		if err != nil {
			log.Error("db.Query:", err)
			return err
		}

		infos = append(infos, &info)
	}

	r.HardwareInfo = infos
	return nil
}
