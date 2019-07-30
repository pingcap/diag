package task

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
)

type Topology struct {
	ClusterName string `json:"cluster_name"`
	Hosts       []struct {
		Ip         string `json:"ip"`
		Components []struct {
			Name string `json:"name"`
			Port string `json:"port"`
		} `json:"components"`
	} `json:"hosts"`
}

type ParseTopologyTask struct {
	BaseTask
}

func ParseTopology(base BaseTask) Task {
	return &ParseTopologyTask{base}
}

func (t *ParseTopologyTask) Run() error {
	content, err := ioutil.ReadFile(path.Join(t.src, "topology.json"))
	if err != nil {
		log.Error("read file: ", err)
		return err
	}

	if err = json.Unmarshal(content, &t.data.topology); err != nil {
		log.Error("unmarshal: ", err)
		return err
	}

	return nil
}
