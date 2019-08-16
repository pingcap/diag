package basic

import (
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/envs"
	"github.com/pingcap/tidb-foresight/analyzer/input/meta"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type saveBasicInfoTask struct{}

func SaveBasicInfo() *saveBasicInfoTask {
	return &saveBasicInfoTask{}
}

// Save cluster basic info to database, the basic info is from the args passed to collector,
// the meta info collected and the env variables api set for collector.
func (t *saveBasicInfoTask) Run(c *boot.Config, e *envs.Env, m *boot.Model, meta *meta.Meta) {
	clusterCreateTime, err := time.Parse(time.RFC3339, e.Get("CLUSTER_CREATE_TIME"))
	if err != nil {
		log.Error("parse cluster create time from env:", err)
	}
	info := &model.BasicInfo{
		InspectionId:      c.InspectionId,
		ClusterName:       meta.ClusterName,
		ClusterCreateTime: clusterCreateTime,
		InspectTime:       time.Unix(int64(meta.InspectTime), 0),
		TidbAlive:         meta.TidbCount,
		TikvAlive:         meta.TikvCount,
		PdAlive:           meta.PdCount,
	}
	if err := m.InsertInspectionBasicInfo(info); err != nil {
		log.Error("insert basic info:", err)
		return
	}
}
