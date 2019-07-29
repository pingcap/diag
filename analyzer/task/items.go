package task

import (
	log "github.com/sirupsen/logrus"
)

const (
	ITEM_BASIC = "basic"
	ITEM_DBINFO = "dbinfo"
	ITEM_METRIC = "metric"
	ITEM_CONFIG = "config"
	ITEM_PROFILE = "profile"
	ITEM_LOG = "log"
)

type SaveItemsTask struct {
	BaseTask
}

func SaveItems(base BaseTask) Task {
	return &SaveItemsTask {base}
}

func (t *SaveItemsTask) Run() error {
	items := []string{
		ITEM_BASIC, ITEM_DBINFO, ITEM_METRIC, ITEM_CONFIG, ITEM_PROFILE, ITEM_LOG,
	}
	for _, item := range items {
		status := "none"
		message := ""
		if t.data.args.Collect(item) {
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