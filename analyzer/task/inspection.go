package task

import (
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type SaveInspectionTask struct {
	BaseTask
}

func SaveInspection(base BaseTask) Task {
	return &SaveInspectionTask{base}
}

func (t *SaveInspectionTask) Run() error {
	instance := t.data.args.InstanceId
	instanceName := t.data.topology.ClusterName
	createTime := t.data.meta.CreateTime
	finishTime := t.data.meta.EndTime
	components := map[string][]string{}

	for _, h := range t.data.topology.Hosts {
		for _, c := range h.Components {
			components[c.Name] = append(components[c.Name], h.Ip+":"+c.Port)
		}
	}

	if _, err := t.db.Exec(
		`INSERT INTO inspections(id,instance,instance_name,user,status,type,tidb,tikv,pd,grafana,prometheus,create_t,finish_t,scrape_bt,scrape_et)
		  VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.inspectionId, instance, instanceName, t.data.env["FORESIGHT_USER"], "running", "manual", strings.Join(components["tidb"], ","),
		strings.Join(components["tikv"], ","), strings.Join(components["pd"], ","), strings.Join(components["grafana"], ","),
		strings.Join(components["prometheus"], ","), time.Unix(int64(createTime), 0), time.Unix(int64(finishTime), 0),
		t.data.args.ScrapeBegin, t.data.args.ScrapeEnd,
	); err != nil {
		log.Error("db.Exec: ", err)
		return err
	}

	return nil
}
