package topology

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
)

type parseTopologyTask struct{}

func ParseTopology() *parseTopologyTask {
	return &parseTopologyTask{}
}

// Parse topology.json
func (t *parseTopologyTask) Run(c *boot.Config) *Topology {
	topo := Topology{}

	content, err := ioutil.ReadFile(path.Join(c.Src, "topology.json"))
	if err != nil {
		log.Error("read file:", err)
		return nil
	}

	if err = json.Unmarshal(content, &topo); err != nil {
		log.Error("unmarshal:", err)
		return nil
	}

	return &topo
}
