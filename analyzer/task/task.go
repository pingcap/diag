package task

import (
	"database/sql"

	log "github.com/sirupsen/logrus"
)

type TaskData struct {
	collect  map[string]bool
	topology Topology
	status   ItemStatus
	meta     Meta
	resource Resource
	dbinfo   DBInfo
	matrix   Matrix
	alert	 AlertInfo
	insight  Insight
	dmesg	Dmesg
}

type Task interface {
	Run() error
}

type BaseTask struct {
	inspectionId string
	src          string
	data         *TaskData
	db           *sql.DB
}

func (t *BaseTask) InsertSymptom(status, message, description string) error {
	if _, err := t.db.Exec(
		"INSERT INTO inspection_symptoms(inspection, status, message, description) VALUES(?, ?, ?, ?)",
		t.inspectionId, status, message, description,
	); err != nil {
		log.Error("insert symptom: ", err)
		return err
	}
	return nil
}

func (t *BaseTask) SetStatus(item, status, message, description string) error {
	t.data.status[item] = struct{
		Status string `json:"status"`
		Message string `json:"message"`
	}{status, message}

	return t.InsertSymptom(status, message, description)
}