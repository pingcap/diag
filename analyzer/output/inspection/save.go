package inspection

import (
	"strings"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/input/envs"
	"github.com/pingcap/tidb-foresight/analyzer/input/meta"
	"github.com/pingcap/tidb-foresight/analyzer/input/topology"
	log "github.com/sirupsen/logrus"
)

type saveInspectionTask struct{}

func SaveInspection() *saveInspectionTask {
	return &saveInspectionTask{}
}

// Save inspection main record to database (then the frontend can see it)
func (t *saveInspectionTask) Run(db *boot.DB, c *boot.Config, args *args.Args, topo *topology.Topology, meta *meta.Meta, e *envs.Env) {
	instance := args.InstanceId
	instanceName := topo.ClusterName
	createTime := meta.CreateTime
	finishTime := meta.EndTime
	components := map[string][]string{}

	for _, h := range topo.Hosts {
		for _, c := range h.Components {
			components[c.Name] = append(components[c.Name], h.Ip+":"+c.Port)
		}
	}

	if _, err := db.Exec(
		`REPLACE INTO inspections(id,instance,instance_name,user,status,type,tidb,tikv,pd,grafana,prometheus,create_t,finish_t,scrape_bt,scrape_et)
		  VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.InspectionId, instance, instanceName, e.Get("FORESIGHT_USER"), "running", e.Get("INSPECTION_TYPE"),
		strings.Join(components["tidb"], ","), strings.Join(components["tikv"], ","), strings.Join(components["pd"], ","),
		strings.Join(components["grafana"], ","), strings.Join(components["prometheus"], ","), time.Unix(int64(createTime), 0),
		time.Unix(int64(finishTime), 0), args.ScrapeBegin, args.ScrapeEnd,
	); err != nil {
		log.Error("db.Exec:", err)
		return
	}
}
