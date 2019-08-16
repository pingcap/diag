package report

import (
	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

type NetworkInfo struct {
	NodeIp      string `json:"node_ip"`
	Connections string `json:"connections"`
	Recv        string `json:"recv"`
	Send        string `json:"send"`
	BadSeg      string `json:"bad_seg"`
	Retrans     string `json:"retrans"`
}

func GetNetworkInfo(db db.DB, inspectionId string) ([]*NetworkInfo, error) {
	infos := []*NetworkInfo{}

	rows, err := db.Query(
		`SELECT node_ip, connections, recv, send, bad_seg, retrans FROM inspection_network WHERE inspection = ?`,
		inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return infos, err
	}

	for rows.Next() {
		info := NetworkInfo{}
		err = rows.Scan(&info.NodeIp, &info.Connections, &info.Recv, &info.Send, &info.BadSeg, &info.Retrans)
		if err != nil {
			log.Error("db.Query:", err)
			return infos, err
		}

		infos = append(infos, &info)
	}

	return infos, nil
}
