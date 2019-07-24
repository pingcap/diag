package task

import (
	"time"
	"path"
	"io/ioutil"
	"encoding/json"
	"database/sql"
	log "github.com/sirupsen/logrus"
)

type Meta struct {
	InstanceId string `json:"instance_id"`
	ClusterName string `json:"cluster_name"`
	CreateTime time.Time `json:"create_time"`
	InspectTime time.Time `json:"inspect_time"`
	TidbCount int `json:"tidb_count"`
	TikvCount int `json:"tikv_count"`
	PdCount int `json:"pd_count"`
}

type ParseMetaTask struct {
	BaseTask
}

func ParseMeta(inspectionId string, src string, data *TaskData, db *sql.DB) Task {
	return &ParseMetaTask {BaseTask{inspectionId, src, data, db}}
}

func (t *ParseMetaTask) Run() error {
	if !t.data.collect[ITEM_BASIC] || t.data.status[ITEM_BASIC].Status != "success" {
		return nil
	}

	content, err := ioutil.ReadFile(path.Join(t.src, "meta.json"))
	if err != nil {
		log.Error("read file: ", err)
		return err
	}

	if err = json.Unmarshal(content, &t.data.meta); err != nil {
		log.Error("unmarshal: ", err)
		return err
	}

	return nil
}