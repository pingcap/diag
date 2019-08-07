package task

import (
	log "github.com/sirupsen/logrus"
)

type ClearTask struct {
	BaseTask
}

func Clear(base BaseTask) Task {
	return &ClearTask{base}
}

func (t *ClearTask) Run() error {
	if _, err := t.db.Exec("DELETE FROM inspection_items WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_items: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_symptoms WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_symptoms: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_basic_info WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_basic_info: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_db_info WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_db_info: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_slow_log WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_slow_log: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_network WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_network: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_alerts WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_alerts: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_hardware WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_hardware: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_dmesg WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_dmesg: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM software_version WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete software_version: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM software_config WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete software_config: ", err)
		return err
	}

	if _, err := t.db.Exec("DELETE FROM inspection_resource WHERE inspection = ?", t.inspectionId); err != nil {
		log.Error("delete inspection_resource: ", err)
		return err
	}

	return nil
}
