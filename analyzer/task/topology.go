package task

import (
	"path"
	"encoding/json"
	"io/ioutil"
	"database/sql"
	log "github.com/sirupsen/logrus"
)

type Topology struct {
	ClusterName string `json:"cluster_name"`
	Hosts []struct {
		Ip string `json:"ip"`
		Components []struct {
			Name string `json:"name"`
			Port string `json:"port"`
		} `json:"components"`
	} `json:"hosts"`
}

type ParseTopologyTask struct {
	BaseTask
}

func ParseTopology(inspectionId string, src string, data *TaskData, db *sql.DB) Task {
	return &ParseTopologyTask {BaseTask{inspectionId, src, data, db}}
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