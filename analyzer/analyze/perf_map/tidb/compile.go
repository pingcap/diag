package tidb

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/utils"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type compileDurationChecker struct{}

func checkCompileDuration() *compileDurationChecker {
	return &compileDurationChecker{}
}

func (t *compileDurationChecker) Run(c *boot.Config, m *boot.Model, mtr *metric.Metric, args *args.Args) {
	if duration := t.compileDuration(mtr, c.InspectionId, args.ScrapeBegin, args.ScrapeEnd); duration > 30 {
		status := "error"
		msg := "compile duration exceed 30ms"
		desc := "typically it should be less than 30ms"
		m.InsertSymptom(status, msg, desc)
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{
			RelatedGraph: "TiDB Compile Duration",
			Problem:      utils.NullStringFromString(msg),
			Advise:       desc,
		})
	} else {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "TiDB Compile Duration"})
	}
}

func (t *compileDurationChecker) compileDuration(m *metric.Metric, insp string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(
			`histogram_quantile(0.99, sum(rate(tidb_session_compile_duration_seconds_bucket{inspectionid="%s"}[1m])) by (le))`,
			insp,
		),
		begin, end,
	); err != nil {
		log.Error("query prom for compile duration:", err)
		return 0
	} else {
		return int(v.Max() * 1000) // s to ms
	}
}
