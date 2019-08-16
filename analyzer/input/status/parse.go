package status

import (
	"encoding/json"
	"io/ioutil"
	"path"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type parseStatusTask struct{}

func ParseStatus() *parseStatusTask {
	return &parseStatusTask{}
}

// Parse status.json, which contains the collect status for each collect item.
func (t *parseStatusTask) Run(c *boot.Config, m *boot.Model) *StatusMap {
	status := &StatusMap{}

	content, err := ioutil.ReadFile(path.Join(c.Src, "status.json"))
	if err != nil {
		log.Error("read file:", err)
		m.InsertSymptom("exception", "parse status.json", "contact developer")
		return nil
	}

	if err = json.Unmarshal(content, status); err != nil {
		log.Error("unmarshal:", err)
		m.InsertSymptom("exception", "parse status.json", "contact developer")
		return nil
	}

	return status
}
