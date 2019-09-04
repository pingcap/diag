package tidb

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	log "github.com/sirupsen/logrus"
)

type parseDurationChecker struct{}

func checkParseDuration() *parseDurationChecker {
	return &parseDurationChecker{}
}

func (t *parseDurationChecker) Run(c *boot.Config, m *boot.Model, mtr *metric.Metric, args *args.Args) {
	if duration := t.parseDuration(mtr, c.InspectionId, args.ScrapeBegin, args.ScrapeEnd); duration > 10 {
		status := "error"
		msg := "parse duration exceed 10ms"
		desc := "typically it should be less than 10ms"
		m.InsertSymptom(status, msg, desc)
	}
}

func (t *parseDurationChecker) parseDuration(m *metric.Metric, insp string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(
			`histogram_quantile(0.99, sum(rate(tidb_session_parse_duration_seconds_bucket{inspectionid="%s"}[1m])) by (le))`,
			insp,
		),
		begin, end,
	); err != nil {
		log.Error("query prom for parse duration:", err)
		return 0
	} else {
		return int(v.Max() * 1000) // s to ms
	}
}
