package task

import (
	log "github.com/sirupsen/logrus"
)

type SaveBasicInfoTask struct {
	BaseTask
}

func SaveBasicInfo(base BaseTask) Task {
	return &SaveBasicInfoTask{base}
}

func (t *SaveBasicInfoTask) Run() error {
	if !t.data.args.Collect(ITEM_BASIC) || t.data.status[ITEM_BASIC].Status != "success" {
		return nil
	}

	if _, err := t.db.Exec(
		`INSERT INTO inspection_basic_info(inspection, cluster_name, cluster_create_t, inspect_t, tidb_count, tikv_count, pd_count) 
		VALUES(?, ?, ?, ?, ?, ?, ?)`, t.inspectionId, t.data.meta.ClusterName, t.data.meta.CreateTime, t.data.meta.InspectTime,
		t.data.meta.TidbCount, t.data.meta.TikvCount, t.data.meta.PdCount,
	); err != nil {
		log.Error("db.Exec: ", err)
		return err
	}

	return nil
}
