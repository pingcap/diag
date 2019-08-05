package task

import (
	"encoding/json"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

type ParseEnvTask struct {
	BaseTask
}

func ParseEnv(base BaseTask) Task {
	return &ParseEnvTask{base}
}

func (t *ParseEnvTask) Run() error {
	t.data.env = make(map[string]string)
	f, err := os.Open(path.Join(t.src, "env.json"))
	if err != nil {
		log.Error("open file: ", err)
		return err
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&t.data.env); err != nil {
		log.Error("decode: ", err)
		return t.SetStatus(ITEM_METRIC, "exception", "parse env.json failed", "contact developer")
	}

	return nil
}
