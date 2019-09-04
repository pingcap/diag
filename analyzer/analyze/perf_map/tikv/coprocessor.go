package tikv

import (
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/input/config"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	log "github.com/sirupsen/logrus"
)

type coprocessorChecker struct{}

func checkCoprocessor() *coprocessorChecker {
	return &coprocessorChecker{}
}

func (t *coprocessorChecker) Run(c *boot.Config, m *boot.Model, tc *config.TiKVConfigInfo, mtr *metric.Metric, args *args.Args) {
	for inst, cfg := range *tc {
		cpu := t.cpu(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		threadNum := cfg.ReadPool.Coprocessor.HighConcurrency + cfg.ReadPool.Coprocessor.NormalConcurrency + cfg.ReadPool.Coprocessor.LowConcurrency
		if cpu > threadNum*90 {
			status := "error"
			msg := fmt.Sprintf("cpu usage of coprocessor exceed 90%% on node %s", inst)
			desc := "The CPU usage should be less than concurrency * 90%"
			m.InsertSymptom(status, msg, desc)
		}
		duration := t.duration(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		if duration > 20 {
			status := "error"
			msg := fmt.Sprintf(".99 coprocessor wait duration exceed 50ms on node %s", inst)
			desc := "It may because coprocessor thread pool is busy."
			m.InsertSymptom(status, msg, desc)
		}
	}
}

func (t *coprocessorChecker) cpu(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`sum(rate(tikv_thread_cpu_seconds_total{inspectionid="%s",instance="%s",name=~"coprocessor_.*"}[5m]))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv coprocessor cpu usage:", err)
		return 0
	} else {
		return int(v.Max() * 100)
	}
}

func (t *coprocessorChecker) duration(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{inspectionid="%s",instance="%s"}[5m])) by (le))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv coprocessor wait duration:", err)
		return 0
	} else {
		return int(v.Max() * 1000)
	}
}
