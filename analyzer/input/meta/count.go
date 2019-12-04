package meta

import (
	"fmt"
	"github.com/pingcap/tidb-foresight/analyzer/manager/nilmap"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	log "github.com/sirupsen/logrus"
)

type countComponentTask struct{}

func CountComponent() *countComponentTask {
	return &countComponentTask{}
}

type CountComponentsDone struct{}

func init() {
	nilmap.TolerateRegisterStruct(CountComponentsDone{})
}

// Query component count from prometheus, which is filled with metric collector collected.
func (t *countComponentTask) Run(c *boot.Config, meta *Meta, args *args.Args, mtr *metric.Metric) *CountComponentsDone {
	meta.TidbCount = t.count(mtr, c.InspectionId, "tidb", args.ScrapeEnd)
	meta.TikvCount = t.count(mtr, c.InspectionId, "tikv", args.ScrapeEnd)
	meta.PdCount = t.count(mtr, c.InspectionId, "pd", args.ScrapeEnd)

	return &CountComponentsDone{}
}

func (t *countComponentTask) count(m *metric.Metric, inspectionId, component string, st time.Time) int {
	v, err := m.Query(
		fmt.Sprintf(`count(probe_success{group="%s", inspectionid="%s"} == 1)`, component, inspectionId),
		st,
	)
	if err != nil {
		log.Warn("query prom:", err)
		return 0
	} else {
		return int(v)
	}
}
