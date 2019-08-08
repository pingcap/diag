package task

import (
	"database/sql"
	"path"

	log "github.com/sirupsen/logrus"
)

type TaskData struct {
	env      map[string]string
	args     Args
	topology Topology
	status   ItemStatus
	meta     Meta
	resource Resource
	dbinfo   DBInfo
	matrix   Matrix
	alert    AlertInfo
	insight  Insight
	dmesg    Dmesg
}

type Task interface {
	Run() error
}

type BaseTask struct {
	inspectionId string
	home         string
	bin          string
	src          string
	logs         string
	profile      string
	data         *TaskData
	db           *sql.DB
}

func NewTask(inspectionId, home string, data *TaskData, db *sql.DB) BaseTask {
	return BaseTask{
		inspectionId,
		home,
		path.Join(home, "bin"),
		path.Join(home, "inspection", inspectionId),
		path.Join(home, "remote-log", inspectionId),
		path.Join(home, "profile", inspectionId),
		data,
		db,
	}
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
	t.data.status[item] = struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}{status, message}

	return t.InsertSymptom(status, message, description)
}

type Symptom struct {
	status string
	message string
	description string
}

func (t *BaseTask) InsertSymptoms(symptoms []Symptom) error {
	for _, item := range symptoms{
		if _, err := t.db.Exec(
			"INSERT INTO inspection_symptoms(inspection, status, message, description) VALUES(?, ?, ?, ?)",
			t.inspectionId, item.status, item.message, item.description,
		); err != nil {
			log.Error("insert symptom: ", err)
			return err
		}
	}
	return nil
}