package topology

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/manager/nilmap"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type parseStatusTask struct{}

func ParseStatus() *parseStatusTask {
	return &parseStatusTask{}
}

// ParseStatusDone is a fake struct for Parse status.
type ParseStatusDone struct{}

func init() {
	nilmap.TolerateRegisterStruct(ParseStatusDone{})
}

func (t *parseStatusTask) Run(c *boot.Config, topo *model.Topology, args *args.Args, m *metric.Metric) *ParseStatusDone {
	parseDone := ParseStatusDone{}
	for i, host := range topo.Hosts {
		for j, comp := range host.Components {
			if t.online(m, c.InspectionId, comp.Name, host.Ip, comp.Port, args.ScrapeEnd) {
				topo.Hosts[i].Components[j].Status = "online"
			} else {
				topo.Hosts[i].Components[j].Status = "offline"
			}
		}
	}
	return &parseDone
}

func (t *parseStatusTask) online(m *metric.Metric, inspectionId, component, ip, port string, st time.Time) bool {
	if component == "prometheus" {
		// obviously it's online
		return true
	}

	// Using prometheus to query if the service is online.
	v, err := m.Query(
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
		return int(v) != 0
	}
}
