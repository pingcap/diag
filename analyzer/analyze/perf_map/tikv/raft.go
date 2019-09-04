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

type rocksDBRaftChecker struct{}

func checkRocksDBRaft() *rocksDBRaftChecker {
	return &rocksDBRaftChecker{}
}

func (t *rocksDBRaftChecker) Run(c *boot.Config, m *boot.Model, tc *config.TiKVConfigInfo, mtr *metric.Metric, args *args.Args) {
	for inst := range *tc {
		duration := t.duration(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		if duration > 50 {
			status := "error"
			msg := fmt.Sprintf(".99 raftstore append log duration exceed 50ms on node %s", inst)
			desc := "maybe the disk is busy."
			m.InsertSymptom(status, msg, desc)
		}
	}
}

func (t *rocksDBRaftChecker) duration(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{inspectionid="%s",instance="%s"}[5m])) by (le))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv raftstore append log duration:", err)
		return 0
	} else {
		return int(v.Max() * 1000)
	}
}
