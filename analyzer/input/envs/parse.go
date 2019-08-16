package envs

import (
	"encoding/json"
	"os"
	"path"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type parseEnvTask struct{}

func ParseEnvs() *parseEnvTask {
	return &parseEnvTask{}
}

// Parse env.json, it's looks like:
//	{
//		"CLUSTER_CREATE_TIME": "2019-08-07T11:21:22Z",
//		"INSPECTION_TYPE": "manual",
//		"FORESIGHT_USER": "foresight"
//	}
func (t *parseEnvTask) Run(c *boot.Config, m *boot.Model) *Env {
	e := &Env{}
	f, err := os.Open(path.Join(c.Src, "env.json"))
	if err != nil {
		log.Error("open file:", err)
		m.InsertSymptom("exception", "parse env.json", "contact developer")
		return nil
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(e); err != nil {
		log.Error("decode:", err)
		m.InsertSymptom("exception", "parse env.json", "contact developer")
		return nil
	}

	return e
}
