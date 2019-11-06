package tidb

import (
	"fmt"
	"github.com/pingcap/tidb-foresight/utils"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type tsoRPCDurationChecker struct{}

func checkTso() *tsoRPCDurationChecker {
	return &tsoRPCDurationChecker{}
}

func (t *tsoRPCDurationChecker) Run(c *boot.Config, m *boot.Model, mtr *metric.Metric, args *args.Args) {
	if statements := t.tsoRPCDuration(mtr, c.InspectionId, args.ScrapeBegin, args.ScrapeEnd); statements > 30 {
		status := "error"
		msg := "the latency of tidb and pd exceed 30ms"
		desc := "check the network between tidb and pd"
		m.InsertSymptom(status, msg, desc)
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{
			RelatedGraph: "PD TSO RPC Duration",
			Problem:      utils.NullStringFromString(msg),
			Advise:       desc,
		})
	} else {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "PD TSO RPC Duration"})
	}
}

func (t *tsoRPCDurationChecker) tsoRPCDuration(m *metric.Metric, insp string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(
			`histogram_quantile(0.99, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{inspectionid="%s",type="tso"}[5m])) by (le))`,
			insp,
		),
		begin, end,
	); err != nil {
		log.Error("query prom for pd tso duration:", err)
		return 0
	} else {
		return int(v.Max())
	}
}
