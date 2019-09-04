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

type storageChecker struct{}

func checkStorage() *storageChecker {
	return &storageChecker{}
}

func (t *storageChecker) Run(c *boot.Config, m *boot.Model, tc *config.TiKVConfigInfo, mtr *metric.Metric, args *args.Args) {
	for inst, cfg := range *tc {
		cpu := t.cpu(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		threadNum := cfg.ReadPool.Storage.HighConcurrency + cfg.ReadPool.Storage.NormalConcurrency + cfg.ReadPool.Storage.LowConcurrency
		if cpu > threadNum*90 {
			status := "error"
			msg := fmt.Sprintf("cpu usage of storage read exceed 90%% on node %s", inst)
			desc := "The CPU usage should be less than concurrency * 90%"
			m.InsertSymptom(status, msg, desc)
		}
	}
}

func (t *storageChecker) cpu(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`sum(rate(tikv_thread_cpu_seconds_total{inspectionid="%s",instance="%s",name=~"store_read.*"}[5m]))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv storage read cpu usage:", err)
		return 0
	} else {
		return int(v.Max() * 100)
	}
}
