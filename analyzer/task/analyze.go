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
	if err := t.CheckIndex(); err != nil {
		return err
	}
	if err := t.CheckResource(); err != nil {
		return err
	}
	if err := t.CheckAlert(); err != nil {
		return err
	}
	if err := t.CheckSlowQuery(); err != nil {
		return err
	}
	if err := t.CheckSoftwareVersion(); err != nil {
		return err
	}
	if err := t.CheckSoftwareConfig(); err != nil {
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
	var symptoms []Symptom
	for rows.Next() {
		err := rows.Scan(&db, &table)
		if err != nil {
			return err
		}
		symptoms = append(symptoms, Symptom{
			status:      "error",
			message:     fmt.Sprintf("table %s missing index in databse %s", db, table),
			description: "please add index for the table",
		})
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return t.InsertSymptoms(symptoms)
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
	var symptoms []Symptom
	for rows.Next() {
		err := rows.Scan(&resource, &duration)
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("%s Resource utilization/%s exceeds %d%%",
			resource, duration, ResourceThreshold)
		symptoms = append(symptoms, Symptom{
			status:      "error",
			message:     msg,
			description: "please increase resources appropriately",
		})
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return t.InsertSymptoms(symptoms)
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
	msg := "alert information currently exists in the cluster"
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
	msg := "no slow logs were collected in the cluster"
	err = t.InsertSymptom("error", msg, "")
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
	msg := "software versions are not identical"
	desc := "make sure all software have the same version"
	err = t.InsertSymptom("error", msg, desc)
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
	msg := "software config is not identical"
	desc := "make sure all software have the same config"
	err = t.InsertSymptom("error", msg, desc)
	if err != nil {
		return err
	}
	return nil
}
