package ntp

import (
	"fmt"
	"math"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/insight"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

const NTP_THRESHOLD = 500.0

type saveNtpInfoTask struct{}

func SaveNtpInfo() *saveNtpInfoTask {
	return &saveNtpInfoTask{}
}

// Save hardware info to database, the hardware info comes from insight collector
func (t *saveNtpInfoTask) Run(m *boot.Model, c *boot.Config, insight *insight.Insight) {
	for _, insight := range *insight {
		offset := utils.NewTagdFloat64(insight.Ntp.Offset, nil)
		if math.Abs(offset.GetValue()) > NTP_THRESHOLD {
			offset.SetTag("status", "abnormal")
			offset.SetTag("message", fmt.Sprintf("ntp offset of node %s exceeded the threshold (500ms)", insight.NodeIp))
		}

		if err := m.InsertInspectionNtpInfo(&model.NtpInfo{
			InspectionId: c.InspectionId,
			NodeIp:       insight.NodeIp,
			Offset:       offset,
		}); err != nil {
			log.Error("insert ntp info:", err)
			return
		}
	}
}
