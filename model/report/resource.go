package report

import (
	log "github.com/sirupsen/logrus"
)

type ResourceInfo struct {
	Name     string  `json:"resource"`
	Duration string  `json:"duration"`
	Value    float64 `json:"value"`
}

func (r *Report) loadResourceInfo() error {
	if !r.itemReady("metric") {
		return nil
	}

	rows, err := r.db.Query(
		`SELECT resource, duration, value from inspection_resource WHERE inspection = ?`,
		r.inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return err
	}

	resources := []*ResourceInfo{}
	for rows.Next() {
		info := ResourceInfo{}
		err = rows.Scan(&info.Name, &info.Duration, &info.Value)
		if err != nil {
			log.Error("db.Query:", err)
			return err
		}

		resources = append(resources, &info)
	}

	r.ResourceInfo = resources
	return nil
}
