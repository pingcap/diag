package meta

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type parseMetaTask struct{}

func ParseMeta() *parseMetaTask {
	return &parseMetaTask{}
}

// Parse meta information from meta.json
func (t *parseMetaTask) Run(c *boot.Config, m *boot.Model) *Meta {
	content, err := ioutil.ReadFile(path.Join(c.Src, "meta.json"))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("read file:", err)
		}
		return nil
	}

	meta := &Meta{}
	if err = json.Unmarshal(content, meta); err != nil {
		log.Error("unmarshal:", err)
		m.InsertSymptom("exception", "parse meta.json", "contact developer")
		return nil
	}

	return meta
}
