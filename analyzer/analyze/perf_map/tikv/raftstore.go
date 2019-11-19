package tikv

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

type raftstoreChecker struct{}

func checkRaftstore() *raftstoreChecker {
	return &raftstoreChecker{}
}

func (t *raftstoreChecker) Run(c *boot.Config, m *boot.Model, tc *config.TiKVConfigInfo, mtr *metric.Metric, args *args.Args) {
	cpuAbnormal := false
	durationAbnormal := false
	for inst, cfg := range *tc {
		cpu := t.cpu(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		if cpu > cfg.RaftStore.StorePoolSize*85 {
			status := "error"
			msg := fmt.Sprintf("cpu usage of raftstore exceed 85%% on node %s", inst)
			desc := "The CPU usage of the raftstore thread pool should be less than store-pool-size * 85%."
			m.InsertSymptom(status, msg, desc)
			m.AddProblem(c.InspectionId, &model.EmphasisProblem{
				RelatedGraph: "Raft store CPU",
				Problem:      utils.NullStringFromString(msg),
				Advise:       desc,
			})
			cpuAbnormal = true
		}
		duration := t.duration(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		if duration > 20 {
			status := "error"
			msg := fmt.Sprintf(".99 propose wait duration exceed 50ms on node %s", inst)
			desc := "It may because append raft log is slow or the CPU of raftstore is high."
			m.InsertSymptom(status, msg, desc)
			m.AddProblem(c.InspectionId, &model.EmphasisProblem{
				RelatedGraph: "99% append log duration",
				Problem:      utils.NullStringFromString(msg),
				Advise:       desc,
			})
			durationAbnormal = true
		}
	}
	if !cpuAbnormal {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "Raft store CPU"})
	}
	if !durationAbnormal {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "99% append log duration"})
	}
}

func (t *raftstoreChecker) cpu(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`sum(rate(tikv_thread_cpu_seconds_total{inspectionid="%s",instance="%s",name=~"raftstore_.*"}[5m]))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv raftstore cpu usage:", err)
		return 0
	} else {
		return int(v.Max() * 100)
	}
}

func (t *raftstoreChecker) duration(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tikv_raftstore_request_wait_time_duration_secs_bucket{inspectionid="%s",instance="%s"}[5m])) by (le))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv raftstore wait duration:", err)
		return 0
	} else {
		return int(v.Max() * 1000)
	}
}
