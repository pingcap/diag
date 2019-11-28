package tidb

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/input/config"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type tokenChecker struct{}

func checkTokenLimit() *tokenChecker {
	return &tokenChecker{}
}

func (t *tokenChecker) Run(c *boot.Config, m *boot.Model, args *args.Args, mtr *metric.Metric, ci *ConnectionCountInfo, tc *config.TiDBConfigInfo) {
	for inst, cfg := range *tc {
		if cfg.TokenLimit < (*ci)[inst] {
			status := "error"
			msg := fmt.Sprintf("token limit should be larger than the connection count on instance %s", inst)
			desc := "it may take a significant time to get a token if token limit is less than connection count"
			m.InsertSymptom(status, msg, desc)
		}
	}

	d := t.getTokenDuration(mtr, c.InspectionId, args.ScrapeBegin, args.ScrapeEnd)
	if d > 2 {
		status := "error"
		msg := "get token duration exceed 2us"
		desc := `typically the "Get Token Duration" should be less than 2us, check if server is busy`
		m.InsertSymptom(status, msg, desc)
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{
			RelatedGraph: "99% Get Token Duration",
			Problem:      utils.NullStringFromString(msg),
			Advise:       desc,
		})
	} else {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "99% Get Token Duration"})
	}
}

func (*tokenChecker) getTokenDuration(m *metric.Metric, insp string, begin, end time.Time) float64 {
	if v, err := m.QueryRange(
		fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tidb_server_get_token_duration_seconds_bucket{inspectionid="%s"}[1m])) by (le))`, insp),
		begin, end,
	); err != nil {
		log.Error("query prom for heap size:", err)
		return 0
	} else {
		return v.Max()
	}
}
