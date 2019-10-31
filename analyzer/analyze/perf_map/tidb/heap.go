package tidb

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/input/config"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

const GB = 1024 * 1024 * 1024

type heapChecker struct{}

func checkHeapMemory() *heapChecker {
	return &heapChecker{}
}

func (t *heapChecker) Run(c *boot.Config, m *boot.Model, tc *config.TiDBConfigInfo, mtr *metric.Metric, args *args.Args) {
	abnormal := false
	for inst, cfg := range *tc {
		size := t.heap(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		if cfg.TxnLocalLatches.Enabled {
			if size > 3*GB {
				status := "error"
				msg := fmt.Sprintf("heap memory usage exceed 3GB on instance %s", inst)
				desc := "the memory usage should be less than 3GB when local-latch is enabled for OLTP workloads"
				m.InsertSymptom(status, msg, desc)
				m.AddProblem(c.InspectionId, &model.EmphasisProblem{
					RelatedGraph: "TiDB Heap Memory Usage",
					Problem:      sql.NullString{msg, true},
					Advise:       desc,
				})
				abnormal = true
			}
		} else {
			if size > 1*GB {
				status := "error"
				msg := fmt.Sprintf("heap memory usage exceed 1GB on instance %s", inst)
				desc := "the memory usage should be less than 1GB when local-latch is enabled for OLTP workloads"
				m.InsertSymptom(status, msg, desc)
				m.AddProblem(c.InspectionId, &model.EmphasisProblem{
					RelatedGraph: "TiDB Heap Memory Usage",
					Problem:      sql.NullString{msg, true},
					Advise:       desc,
				})
				abnormal = true
			}
		}
	}
	if !abnormal {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "TiDB Heap Memory Usage"})
	}
}

func (t *heapChecker) heap(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`go_memstats_heap_inuse_bytes{inspectionid="%s", instance="%s", job=~"tidb.*"}`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for heap size:", err)
		return 0
	} else {
		return int(v.Max())
	}
}
