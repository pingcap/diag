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

type grpcChecker struct{}

func checkGrpc() *grpcChecker {
	return &grpcChecker{}
}

func (t *grpcChecker) Run(c *boot.Config, m *boot.Model, tc *config.TiKVConfigInfo, mtr *metric.Metric, args *args.Args) {
	abnormal := false
	for inst, cfg := range *tc {
		cpu := t.cpu(mtr, c.InspectionId, inst, args.ScrapeBegin, args.ScrapeEnd)
		if cpu > cfg.Server.GrpcConcurrency*90 {
			status := "error"
			msg := fmt.Sprintf("cpu usage of gRPC exceed 90%% on node %s", inst)
			desc := "The CPU usage of the gRPC thread pool should be less than grpc-concurrency * 90%"
			m.InsertSymptom(status, msg, desc)
			m.AddProblem(c.InspectionId, &model.EmphasisProblem{
				RelatedGraph: "gRPC poll CPU",
				Problem:      utils.NullStringFromString(msg),
				Advise:       desc,
			})
			abnormal = true
		}
	}
	if !abnormal {
		m.AddProblem(c.InspectionId, &model.EmphasisProblem{RelatedGraph: "gRPC poll CPU"})
	}
}

func (t *grpcChecker) cpu(m *metric.Metric, insp, inst string, begin, end time.Time) int {
	if v, err := m.QueryRange(
		fmt.Sprintf(`sum(rate(tikv_thread_cpu_seconds_total{inspectionid="%s", instance=~"%s", name=~"grpc.*"}[5m]))`, insp, inst),
		begin, end,
	); err != nil {
		log.Error("query prom for tikv grpc cpu usage:", err)
		return 0
	} else {
		return int(v.Max() * 100)
	}
}
