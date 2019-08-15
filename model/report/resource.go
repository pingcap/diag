package report

import (
	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

type ResourceInfo struct {
	Name     string  `json:"resource"`
	Duration string  `json:"duration"`
	Value    float64 `json:"value"`
}

func GetResourceInfo(db db.DB, inspectionId string) ([]*ResourceInfo, error) {
	resources := []*ResourceInfo{}

	rows, err := db.Query(
		`SELECT resource, duration, value from inspection_resource WHERE inspection = ?`,
		inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return resources, err
	}

	for rows.Next() {
		info := ResourceInfo{}
		err = rows.Scan(&info.Name, &info.Duration, &info.Value)
		if err != nil {
			log.Error("db.Query:", err)
			return resources, err
		}

		resources = append(resources, &info)
	}

	return resources, nil
}
