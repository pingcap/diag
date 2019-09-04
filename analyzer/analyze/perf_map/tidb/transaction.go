package tidb

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	log "github.com/sirupsen/logrus"
)

type transactionChecker struct{}

func checkTransaction() *transactionChecker {
	return &transactionChecker{}
}

func (t *transactionChecker) Run(c *boot.Config, m *boot.Model, mtr *metric.Metric, args *args.Args) {
	if statements := t.statementNum(mtr, c.InspectionId, args.ScrapeBegin, args.ScrapeEnd); statements > 500 {
		status := "error"
		msg := "transaction statement number exceed 500"
		desc := "typically it should be less than 500"
		m.InsertSymptom(status, msg, desc)
	}
	if retry := t.retryNum(mtr, c.InspectionId, args.ScrapeEnd, args.ScrapeBegin); retry > 3 {
		status := "error"
		msg := "transaction retry number exceed 3"
		desc := "there are many write-write conflicts"
		m.InsertSymptom(status, msg, desc)
	}
}

func (t *transactionChecker) statementNum(m *metric.Metric, insp string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(
			`histogram_quantile(1.0, sum(rate(tidb_session_transaction_statement_num_bucket{inspectionid="%s"}[5m])) by (le))`,
			insp,
		),
		begin, end,
	); err != nil {
		log.Error("query prom for transaction number:", err)
		return 0
	} else {
		return int(v.Max())
	}
}

func (t *transactionChecker) retryNum(m *metric.Metric, insp string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(
			`histogram_quantile(1.0, sum(rate(tidb_session_retry_num_bucket{inspectionid="%s"}[5m])) by (le))`,
			insp,
		),
		begin, end,
	); err != nil {
		log.Error("query prom for retry number:", err)
		return 0
	} else {
		return int(v.Max())
	}
}
