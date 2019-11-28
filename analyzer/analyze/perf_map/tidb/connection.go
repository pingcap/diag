package tidb

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type ConnectionCountInfo map[string]int

type connectionChecker struct{}

func checkConnection() *connectionChecker {
	return &connectionChecker{}
}

func (ck *connectionChecker) Run(
	c *boot.Config,
	m *boot.Model,
	topo *model.Topology,
	args *args.Args,
	mtr *metric.Metric) *ConnectionCountInfo {

	info := ConnectionCountInfo{}
	counts := []int{}
	sum := 0
	abnormal := false
	for _, host := range topo.Hosts {
		for _, comp := range host.Components {
			// only check tidb connections here
			if comp.Name != "tidb" {
				continue
			}
			count := ck.count(mtr, c.InspectionId, host.Ip, comp.Port, args.ScrapeBegin, args.ScrapeEnd)
			if count > 500 {
				status := "error"
				msg := fmt.Sprintf("the suggested connection count should be less than 500 under OLTP workload on node %s", host)
				desc := "maybe you should add more tidb server"
				m.InsertSymptom(status, msg, desc)
				m.AddProblem(c.InspectionId, &model.EmphasisProblem{
					RelatedGraph: "TiDB Connection count",
					Problem:      utils.NullStringFromString(msg),
					Advise:       desc,
				})
				abnormal = true
			}
			counts = append(counts, count)
			sum += count
			info[fmt.Sprintf("%s:%s", host.Ip, comp.Port)] = count
		}
	}

	// only when the total connection is more than 100 we check the balance
	if sum > 100 {
		max := counts[0]
		min := counts[0]
		for _, count := range counts {
			if count > max {
				max = count
			}
			if count < min {
				min = count
			}
		}
		if float64(max-min)/float64(max) > 0.1 {
			status := "error"
			msg := "the number of connections is not balanced between multiple tidb-server instances"
			desc := "please check if your load balancer is working properly"
			m.InsertSymptom(status, msg, desc)
			m.AddProblem(c.InspectionId, &model.EmphasisProblem{
				RelatedGraph: "TiDB Connection Count",
				Problem:      utils.NullStringFromString(msg),
				Advise:       desc,
			})
			abnormal = true
		}
	}

	if !abnormal {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "TiDB Connection Count"})
	}

	return &info
}

func (*connectionChecker) count(m *metric.Metric, inspId, ip, port string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(
			`tidb_server_connections{inspectionid="%s", instance="%s:%s"}`,
			inspId,
			ip,
			port,
		),
		begin,
		end,
	); err != nil {
		log.Error("query prom for connection count:", err)
		return 0
	} else {
		return int(v.Max())
	}
}
