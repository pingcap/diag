package meta

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	"github.com/pingcap/tidb-foresight/analyzer/utils"
	log "github.com/sirupsen/logrus"
)

type countComponentTask struct{}

func CountComponent() *countComponentTask {
	return &countComponentTask{}
}

// Query component count from prometheus, which is filled with metric collector collected.
func (t *countComponentTask) Run(c *boot.Config, meta *Meta, args *args.Args, m *metric.Metric /* DO NOT REMOVE ME */) *Meta {
	meta.TidbCount = t.count(c.InspectionId, "tidb", args.ScrapeEnd)
	meta.TikvCount = t.count(c.InspectionId, "tikv", args.ScrapeEnd)
	meta.PdCount = t.count(c.InspectionId, "pd", args.ScrapeEnd)

	return meta
}

func (t *countComponentTask) count(inspectionId, component string, st time.Time) int {
	v, err := utils.QueryProm(
		fmt.Sprintf(`count(probe_success{group="%s", inspectionid="%s"} == 1)`, component, inspectionId),
		st,
	)
	if err != nil {
		log.Warn("query prom:", err)
		return 0
	} else {
		return int(*v)
	}
}
