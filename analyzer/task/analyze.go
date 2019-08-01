package task

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type AnalyzeTask struct {
	BaseTask
}

func Analyze(base BaseTask) Task {
	return &AnalyzeTask{base}
}

func (t *AnalyzeTask) Run() error {
	if _, err := t.db.Exec(
		`UPDATE inspection_items SET status = 'success' WHERE inspection = ? AND status = 'running'`,
		t.inspectionId,
	); err != nil {
		log.Error("db.Exec: ", err)
		return err
	}

	return nil
}

func (t *AnalyzeTask) CheckIndex() error {
	rows, err := t.db.Query(
		`SELECT db, tb FROM inspection_db_info WHERE idx = 0 AND inspection = ?`,
		t.inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return err
	}
	var db, table string
	for rows.Next() {
		err := rows.Scan(&db, &table)
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("table %s missing index in databse %s", db, table)
		err = t.InsertSymptom("error", msg, "")
		if err != nil {
			return err
		}
	}
	return nil
}

const ResourceThreshold = 20

func (t *AnalyzeTask) CheckResource() error {
	rows, err := t.db.Query(
		`SELECT resource, duration FROM inspection_resource WHERE inspection = ? AND value > ?`,
		t.inspectionId, ResourceThreshold,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return err
	}
	var resource, duration string
	for rows.Next() {
		err := rows.Scan(&resource, &duration)
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("%s Resource utilization/%s exceeds %d%%, Please increase resources appropriately.",
			resource, duration, ResourceThreshold)
		err = t.InsertSymptom("warning", msg, "")
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *AnalyzeTask) CheckAlert() error {
	var count int
	err := t.db.QueryRow(
		`SELECT count(*) FROM inspection_alerts WHERE inspection = ?`,
		t.inspectionId,
	).Scan(&count)
	if err != nil {
		log.Error("db.QueryRow: ", err)
		return err
	}
	if count == 0 {
		return nil
	}
	msg := "Alarm information currently exists in the cluster"
	err = t.InsertSymptom("warning", msg, "")
	if err != nil {
		return err
	}
	return nil
}

func (t *AnalyzeTask) CheckSlowQuery() error {
	var count int
	err := t.db.QueryRow(
		`SELECT count(*) FROM inspection_slow_log WHERE inspection = ?`,
		t.inspectionId,
	).Scan(&count)
	if err != nil {
		log.Error("db.QueryRow: ", err)
		return err
	}
	if count != 0 {
		return nil
	}
	msg := "No slow logs were collected in the cluster"
	err = t.InsertSymptom("warning", msg, "")
	if err != nil {
		return err
	}
	return nil
}

func (t *AnalyzeTask) CheckSoftwareVersion() error {
	var count int
	err := t.db.QueryRow(
		`SELECT COUNT(DISTINCT(version)) FROM software_version WHERE inspection = ? GROUP BY component`,
		t.inspectionId,
	).Scan(&count)
	if err != nil {
		log.Error("db.QueryRow: ", err)
		return err
	}
	if count == 1 {
		return nil
	}
	msg := "Inconsistent software version information"
	err = t.InsertSymptom("warning", msg, "")
	if err != nil {
		return err
	}
	return nil
}

func (t *AnalyzeTask) CheckSoftwareConfig() error {
	var count int
	err := t.db.QueryRow(
		`SELECT COUNT(DISTINCT(config)) FROM software_config WHERE inspection = ? GROUP BY component`,
		t.inspectionId,
	).Scan(&count)
	if err != nil {
		log.Error("db.QueryRow: ", err)
		return err
	}
	if count == 1 {
		return nil
	}
	msg := "Inconsistent software config information"
	err = t.InsertSymptom("warning", msg, "")
	if err != nil {
		return err
	}
	return nil
}
