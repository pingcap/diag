package task

import (
	"time"
	"strings"
	log "github.com/sirupsen/logrus"
)

type SaveInspectionTask struct {
	BaseTask
}

func SaveInspection(base BaseTask) Task {
	return &SaveInspectionTask {base}
}

func (t *SaveInspectionTask) Run() error {
	instance := t.data.meta.InstanceId
	createTime := t.data.meta.CreateTime
	components := map[string][]string{}

	for _, h := range t.data.topology.Hosts {
		for _, c := range h.Components {
			components[c.Name] = append(components[c.Name], h.Ip + ":" + c.Port)
		}
	}

	if _, err := t.db.Exec(
		 `INSERT INTO inspections(id, instance, status, type, tidb, tikv, pd, grafana, prometheus, create_t)
		  VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		  t.inspectionId, instance, "running", "manual", strings.Join(components["tidb"], ","),
		  strings.Join(components["tikv"], ","), strings.Join(components["pd"], ","),
		  strings.Join(components["grafana"], ","), strings.Join(components["prometheus"], ","), time.Unix(int64(createTime), 0),
	); err != nil {
		log.Error("db.Exec: ", err)
		return err
	}

	return nil
}
