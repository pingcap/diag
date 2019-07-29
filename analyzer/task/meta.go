package task

import (
	"fmt"
	"time"
	"path"
	"io/ioutil"
	"encoding/json"

	"github.com/pingcap/tidb-foresight/analyzer/utils"
	log "github.com/sirupsen/logrus"
)

type Meta struct {
	InstanceId string `json:"instance_id"`
	ClusterName string `json:"cluster_name"`
	CreateTime float64 `json:"create_time"`
	InspectTime float64 `json:"inspect_time"`
	TidbCount int `json:"tidb_count"`
	TikvCount int `json:"tikv_count"`
	PdCount int `json:"pd_count"`
}

type ParseMetaTask struct {
	BaseTask
}

func ParseMeta(base BaseTask) Task {
	return &ParseMetaTask {base}
}

func (t *ParseMetaTask) Run() error {
	content, err := ioutil.ReadFile(path.Join(t.src, "meta.json"))
	if err != nil {
		log.Error("read file: ", err)
		return err
	}

	if err = json.Unmarshal(content, &t.data.meta); err != nil {
		log.Error("unmarshal:", err)
		return err
	}

	if !t.data.args.Collect(ITEM_METRIC) || t.data.status[ITEM_METRIC].Status != "success" {
		return nil
	}

	t.data.meta.TidbCount = t.CountComponent("tidb")
	t.data.meta.TikvCount = t.CountComponent("tikv")
	t.data.meta.PdCount = t.CountComponent("pd")

	return nil
}

func (t *ParseMetaTask) CountComponent(component string) int {
	v, err := utils.QueryProm(
		fmt.Sprintf(`count(probe_success{group="%s", inspectionid="%s"} == 1)`, component, t.inspectionId),
		time.Unix(int64(t.data.meta.InspectTime), 0),
	)
	if err != nil || v == nil {
		return 0
	} else {
		return int(*v)	
	}
}
