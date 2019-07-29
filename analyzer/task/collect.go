package task

import (
	"path"
	"strings"
	"io/ioutil"
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

type ParseCollectTask struct {
	BaseTask
}

func ParseCollect(base BaseTask) Task {
	return &ParseCollectTask {base}
}

func (t *ParseCollectTask) Run() error {
	content, err := ioutil.ReadFile(path.Join(t.src, "collect.json"))
	if err != nil {
		log.Error("read file: ", err)
		return err
	}

	obj := struct {
		Collect string `json:"collect"`
	}{}
	if err = json.Unmarshal(content, &obj); err != nil {
		log.Error("unmarshal: ", err)
		return err
	}

	t.data.collect = make(map[string]bool)
	items := strings.Split(obj.Collect, ",")
	for _, item := range items {
		name := strings.Split(item, ":")[0]
		t.data.collect[name] = true
	}

	return nil
}