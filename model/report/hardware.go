package report

import (
	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

type HardwareInfo struct {
	NodeIp  string `json:"node_ip"`
	Cpu     string `json:"cpu"`
	Memory  string `json:"memory"`
	Disk    string `json:"disk"`
	Network string `json:"network"`
}

func GetHardwareInfo(db db.DB, inspectionId string) ([]*HardwareInfo, error) {
	infos := []*HardwareInfo{}

	rows, err := db.Query(
		`SELECT node_ip, cpu, memory, disk, network FROM inspection_hardware WHERE inspection = ?`,
		inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return infos, err
	}

	for rows.Next() {
		info := HardwareInfo{}
		err = rows.Scan(&info.NodeIp, &info.Cpu, &info.Memory, &info.Disk, &info.Network)
		if err != nil {
			log.Error("db.Query:", err)
			return infos, err
		}

		infos = append(infos, &info)
	}

	return infos, nil
}
