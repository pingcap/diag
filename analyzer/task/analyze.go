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
			message:     fmt.Sprintf("table %s missing index in database %s", db, table),
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
	msg := "there are alert information in the cluster"
	desc := "please check the alert information"
	err = t.InsertSymptom("warning", msg, desc)
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
	if count == 0 {
		return nil
	}
	msg := "there are slow logs in the cluster"
	desc := "please check the slow log"
	err = t.InsertSymptom("warning", msg, desc)
	if err != nil {
		return err
	}
	return nil
}

func (t *AnalyzeTask) CheckSoftwareVersion() error {
	rows, err := t.db.Query(
		`SELECT component, COUNT(DISTINCT(version)) FROM software_version WHERE inspection = ? GROUP BY component`,
		t.inspectionId,
	)
	if err != nil {
		log.Error("db.QueryRow: ", err)
		return err
	}
	var name string
	var count int
	var symptoms []Symptom
	for rows.Next() {
		err := rows.Scan(&name, &count)
		if err != nil {
			return err
		}
		if count == 1 {
			continue
		}
		msg := fmt.Sprintf("%s versions are not identical", name)
		desc := fmt.Sprintf("make sure all %s have the same version", name)
		symptoms = append(symptoms, Symptom{
			status:      "error",
			message:     msg,
			description: desc,
		})
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return t.InsertSymptoms(symptoms)
}

func (t *AnalyzeTask) CheckSoftwareConfig() error {
	rows, err := t.db.Query(
		`SELECT component, COUNT(DISTINCT(config)) FROM software_config WHERE inspection = ? GROUP BY component`,
		t.inspectionId,
	)
	if err != nil {
		log.Error("db.QueryRow: ", err)
		return err
	}
	var name string
	var count int
	var symptoms []Symptom
	for rows.Next() {
		err := rows.Scan(&name, &count)
		if err != nil {
			return err
		}
		if count == 1 {
			continue
		}
		msg := fmt.Sprintf("%s config is not identical", name)
		desc := fmt.Sprintf("make sure all %s have the same config", name)
		symptoms = append(symptoms, Symptom{
			status:      "error",
			message:     msg,
			description: desc,
		})
	}
	if err := rows.Close(); err != nil {
		return err
	}
	return t.InsertSymptoms(symptoms)
}
