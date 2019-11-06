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

type kvChecker struct{}

func checkKV() *kvChecker {
	return &kvChecker{}
}

func (t *kvChecker) Run(c *boot.Config, m *boot.Model, mtr *metric.Metric, args *args.Args) {
	if statements := t.lockResolveOPS(mtr, c.InspectionId, args.ScrapeBegin, args.ScrapeEnd); statements > 500 {
		status := "error"
		msg := "lock resolve OPS exceed 500"
		desc := "too many write-write/read-write conflicts"
		m.InsertSymptom(status, msg, desc)
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{
			RelatedGraph: "Lock Resolve OPS",
			Problem:      utils.NullStringFromString(msg),
			Advise:       desc,
		})
	} else {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "Lock Resolve OPS"})
	}

	typs := []string{"txnLockFast", "txnLock", "regionMiss"}
	abnormal := false
	for _, tp := range typs {
		if backoff := t.backoffOPS(mtr, c.InspectionId, args.ScrapeBegin, args.ScrapeEnd, tp); backoff > 500 {
			status := "error"
			msg := fmt.Sprintf("backoff OPS of %s exceed 500", tp)
			desc := "too many times to wait and retry transactions are blocked by locks or region routing has been updated."
			m.InsertSymptom(status, msg, desc)
			m.AddProblem(c.InspectionId, &model.EmphasisProblem{
				RelatedGraph: "KV Backoff OPS",
				Problem:      utils.NullStringFromString(msg),
				Advise:       desc,
			})
			abnormal = true
		}
	}
	if !abnormal {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "KV Backoff OPS"})
	}
}

func (t *kvChecker) lockResolveOPS(m *metric.Metric, insp string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(
			`sum(rate(tidb_tikvclient_lock_resolver_actions_total{inspectionid="%s"}[5m]))`,
			insp,
		),
		begin, end,
	); err != nil {
		log.Error("query prom for lock resolve ops:", err)
		return 0
	} else {
		return int(v.Max())
	}
}

func (t *kvChecker) backoffOPS(m *metric.Metric, insp string, begin, end time.Time, tp string) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(
			`sum(rate(tidb_tikvclient_backoff_total{inspectionid="%s",type="%s"}[5m]))`,
			insp, tp,
		),
		begin, end,
	); err != nil {
		log.Error("query prom for backoff ops:", err)
		return 0
	} else {
		return int(v.Max())
	}
}
