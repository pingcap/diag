package task

import (
	"database/sql"
	log "github.com/sirupsen/logrus"
)

const (
	ITEM_BASIC = "basic"
	ITEM_DBINFO = "dbinfo"
	ITEM_SLOWLOG = "slowlog"
	ITEM_METRIC = "metric"
	ITEM_PROFILE = "profile"
	ITEM_RESOURCE = "resource"
	ITEM_ALERT = "alert"
	ITEM_INSIGHT = "insight"
	ITEM_DMESG = "dmesg"
)

type SaveItemsTask struct {
	BaseTask
}

func SaveItems(inspectionId string, src string, data *TaskData, db *sql.DB) Task {
	return &SaveItemsTask {BaseTask{inspectionId, src, data, db}}
}

func (t *SaveItemsTask) Run() error {
	items := []string{
		ITEM_BASIC, ITEM_DBINFO, ITEM_SLOWLOG, ITEM_METRIC, ITEM_PROFILE, 
		ITEM_RESOURCE, ITEM_ALERT, ITEM_INSIGHT, ITEM_DMESG,
	}
	for _, item := range items {
		status := "none"
		message := ""
		if t.data.collect[item] {
			if t.data.status[item].Status == "success" {
					status = "pending"
			} else {
					status = "exception"
					message = t.data.status[item].Message
			}
		}

		if _, err := t.db.Exec(
			"REPLACE INTO inspection_items(inspection, name, status, message) VALUES(?, ?, ?, ?)",
			t.inspectionId, item, status, message,
		); err != nil {
			log.Error("db.Exec: ", err)
			return err
		}
	}

	return nil
}