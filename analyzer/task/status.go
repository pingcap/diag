package task

import (
	"path"
	"io/ioutil"
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

func ParseStatus(base BaseTask) Task {
	return &ParseStatusTask {base}
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