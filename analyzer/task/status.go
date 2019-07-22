package task

import (
	"path"
	"io/ioutil"
	"database/sql"
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

type ItemStatus map[string]struct {
	Status string `json:"status"`
	Message string `json:"message"`
}

type ParseStatusTask struct {
	BaseTask
}

func ParseStatus(inspectionId string, src string, data *TaskData, db *sql.DB) Task {
	return &ParseStatusTask {BaseTask{inspectionId, src, data, db}}
}

func (t *ParseStatusTask) Run() error {
	content, err := ioutil.ReadFile(path.Join(t.src, "status.json"))
	if err != nil {
		log.Error("read file: ", err)
		return err
	}

	if err = json.Unmarshal(content, &t.data.status); err != nil {
		log.Error("unmarshal: ", err)
		return err
	}

	return nil
}