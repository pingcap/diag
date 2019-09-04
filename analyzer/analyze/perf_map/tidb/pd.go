package tidb

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	log "github.com/sirupsen/logrus"
)

type tsoRPCDurationChecker struct{}

func checkTso() *tsoRPCDurationChecker {
	return &tsoRPCDurationChecker{}
}

func (t *tsoRPCDurationChecker) Run(c *boot.Config, m *boot.Model, mtr *metric.Metric, args *args.Args) {
	if statements := t.tsoRPCDuration(mtr, c.InspectionId, args.ScrapeBegin, args.ScrapeEnd); statements > 500 {
		status := "error"
		msg := "lock resolve OPS exceed 500"
		desc := "too many write-write/read-write conflicts"
		m.InsertSymptom(status, msg, desc)
	}
}

func (t *tsoRPCDurationChecker) tsoRPCDuration(m *metric.Metric, insp string, begin, end time.Time) int {
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
