package task

import (
	log "github.com/sirupsen/logrus"
)

type ClearTask struct {
	BaseTask
}

func Clear(base BaseTask) Task {
	return &ClearTask {base}
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
		log.Error("inspection_slow_log: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_hardware WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("inspection_hardware: ", err)
		return err
	}

	return nil
}