package tikv

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/input/config"
	"github.com/pingcap/tidb-foresight/analyzer/output/metric"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type rocksDBKVChecker struct{}

func checkRocksDBKV() *rocksDBKVChecker {
	return &rocksDBKVChecker{}
}

func (t *rocksDBKVChecker) Run(c *boot.Config, m *boot.Model, tc *config.TiKVConfigInfo, mtr *metric.Metric, args *args.Args) {
	waitAbnormal := false
	applyAbnormal := false
	cpuAbnormal := false
	for inst, cfg := range *tc {
		wait := t.wait(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		if wait > 100 {
			status := "error"
			msg := fmt.Sprintf(".99 apply wait duration exceed 100ms on node %s", inst)
			desc := "it means the apply pool is busy or the write db duration is high"
			m.InsertSymptom(status, msg, desc)
			m.AddProblem(c.InspectionId, &model.EmphasisProblem{
				RelatedGraph: "Apply Wait Duration",
				Problem:      sql.NullString{msg, true},
				Advise:       desc,
			})
			waitAbnormal = true
		}
		apply := t.apply(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		if apply > 100 {
			status := "error"
			msg := fmt.Sprintf(".99 apply log duration exceed 100ms on node %s", inst)
			desc := "maybe the disk is busy."
			m.InsertSymptom(status, msg, desc)
			m.AddProblem(c.InspectionId, &model.EmphasisProblem{
				RelatedGraph: "Apply Log Duration",
				Problem:      sql.NullString{msg, true},
				Advise:       desc,
			})
			applyAbnormal = true
		}
		cpu := t.cpu(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		if cpu > cfg.RaftStore.ApplyPoolSize*90 {
			status := "error"
			msg := fmt.Sprintf("The CPU usage of apply pool exceed 90%% on node %s", inst)
			desc := "the apply pool is busy."
			m.InsertSymptom(status, msg, desc)
			m.AddProblem(c.InspectionId, &model.EmphasisProblem{
				RelatedGraph: "Async Apply CPU",
				Problem:      sql.NullString{msg, true},
				Advise:       desc,
			})
			cpuAbnormal = true
		}
	}
	if !waitAbnormal {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "Apply Wait Duration"})
	}
	if !applyAbnormal {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "Apply Log Duration"})
	}
	if !cpuAbnormal {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "Async Apply CPU"})
	}
}

func (t *rocksDBKVChecker) wait(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_wait_time_duration_secs_bucket{inspectionid="%s",instance="%s"}[5m])) by (le))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv apply wait duration:", err)
		return 0
	} else {
		return int(v.Max() * 1000)
	}
}

func (t *rocksDBKVChecker) apply(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{inspectionid="%s",instance="%s"}[5m])) by (le))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv apply log duration:", err)
		return 0
	} else {
		return int(v.Max() * 1000)
	}
}

func (t *rocksDBKVChecker) cpu(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`sum(rate(tikv_thread_cpu_seconds_total{inspectionid="%s",instance="%s",name=~"apply_[0-9]+"}[1m]))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv async apply CPU:", err)
		return 0
	} else {
		return int(v.Max() * 100)
	}
}
