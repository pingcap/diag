package topology

import (
	"encoding/json"
	"io/ioutil"
	"path"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type parseTopologyTask struct{}

func ParseTopology() *parseTopologyTask {
	return &parseTopologyTask{}
}

// Parse topology.json
func (t *parseTopologyTask) Run(c *boot.Config) *model.Topology {
	topo := model.Topology{}

	content, err := ioutil.ReadFile(path.Join(c.Src, "topology.json"))
	if err != nil {
		log.Error("read file:", err)
		return nil
	}

	if err = json.Unmarshal(content, &topo); err != nil {
		log.Error("unmarshal:", err)
		return nil
	}

	for i, host := range topo.Hosts {
		for j := range host.Components {
			topo.Hosts[i].Components[j].Status = "unknown"
		}
	}
	return &topo
}
