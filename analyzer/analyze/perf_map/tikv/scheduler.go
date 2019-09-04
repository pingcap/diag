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

type schedulerChecker struct{}

func checkScheduler() *schedulerChecker {
	return &schedulerChecker{}
}

func (t *schedulerChecker) Run(c *boot.Config, m *boot.Model, tc *config.TiKVConfigInfo, mtr *metric.Metric, args *args.Args) {
	for inst, cfg := range *tc {
		cpu := t.cpu(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		if cpu > cfg.Storage.SchedulerWorkerPoolSize*90 {
			status := "error"
			msg := fmt.Sprintf("cpu usage of scheduler exceed 90%% on node %s", inst)
			desc := "The CPU usage of the scheduler thread pool should be less than scheduler-worker-pool-size * 90%"
			m.InsertSymptom(status, msg, desc)
		}
		duration := t.duration(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		if duration > 20 {
			status := "error"
			msg := fmt.Sprintf(".99 latch wait duration  exceed 20ms on node %s", inst)
			desc := "it means that the conflicts is high or should enlarge the latch."
			m.InsertSymptom(status, msg, desc)
		}
	}
}

func (t *schedulerChecker) cpu(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`sum(rate(tikv_thread_cpu_seconds_total{inspectionid="%s",instance="%s",name=~"sched_.*"}[5m]))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv scheduler cpu usage:", err)
		return 0
	} else {
		return int(v.Max() * 100)
	}
}

func (t *schedulerChecker) duration(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{inspectionid="%s",instance=~"%s"}[5m])) by (le))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv scheduler latch wait duration:", err)
		return 0
	} else {
		return int(v.Max() * 1000)
	}
}
