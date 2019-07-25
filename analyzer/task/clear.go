package task

import (
	"database/sql"
	log "github.com/sirupsen/logrus"
)

type ClearTask struct {
	BaseTask
}

func Clear(inspectionId string, src string, data *TaskData, db *sql.DB) Task {
	return &ClearTask {BaseTask{inspectionId, src, data, db}}
}

func (t *ClearTask) Run() error {
	if _, err := t.db.Exec("DELETE FROM inspections WHERE id = ?", t.inspectionId); err != nil {
		log.Error("delete inspection: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_items WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_items: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_basic_info WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_basic_info: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_slow_log WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_basic_info: ", err)
		return err
	}

	return nil
}