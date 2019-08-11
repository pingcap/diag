package basic

import (
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/envs"
	"github.com/pingcap/tidb-foresight/analyzer/input/meta"
	log "github.com/sirupsen/logrus"
)

type saveBasicInfoTask struct{}

func SaveBasicInfo() *saveBasicInfoTask {
	return &saveBasicInfoTask{}
}

// Save cluster basic info to database, the basic info is from the args passed to collector,
// the meta info collected and the env variables api set for collector.
func (t *saveBasicInfoTask) Run(c *boot.Config, e *envs.Env, db *boot.DB, m *meta.Meta) {
	if _, err := db.Exec(
		`INSERT INTO inspection_basic_info(inspection, cluster_name, cluster_create_t, inspect_t, tidb_count, tikv_count, pd_count) 
		VALUES(?, ?, ?, ?, ?, ?, ?)`, c.InspectionId, m.ClusterName, e.Get("CLUSTER_CREATE_TIME"), time.Unix(int64(m.InspectTime), 0),
		m.TidbCount, m.TikvCount, m.PdCount,
	); err != nil {
		log.Error("db.Exec:", err)
		return
	}
}
