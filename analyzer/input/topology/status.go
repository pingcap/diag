package topology

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type parseStatusTask struct{}

func ParseStatus() *parseStatusTask {
	return &parseStatusTask{}
}

func (t *parseStatusTask) Run(c *boot.Config, topo *Topology, args *args.Args, m *metric.Metric /* DO NOT REMOVE ME */) {
	for i, host := range topo.Hosts {
		for j, comp := range host.Components {
			if t.online(c.InspectionId, comp.Name, host.Ip, comp.Port, args.ScrapeEnd) {
				topo.Hosts[i].Components[j].Status = "online"
			} else {
				topo.Hosts[i].Components[j].Status = "offline"
			}
		}
	}
}

func (t *parseStatusTask) online(inspectionId, component, ip, port string, st time.Time) bool {
	if component == "prometheus" {
		// obviously he is online
		return true
	}

	v, err := utils.QueryProm(
		fmt.Sprintf(
			`count(probe_success{group="%s", inspectionid="%s", instance="%s:%s"} == 1)`,
			component,
			inspectionId,
			ip,
			port,
		),
		st,
	)
	if err != nil {
		log.Warn("query prom:", err)
		return false
	} else {
		return int(*v) != 0
	}
}
