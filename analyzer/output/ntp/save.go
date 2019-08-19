package ntp

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/insight"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type saveNtpInfoTask struct{}

func SaveNtpInfo() *saveNtpInfoTask {
	return &saveNtpInfoTask{}
}

// Save hardware info to database, the hardware info comes from insight collector
func (t *saveNtpInfoTask) Run(m *boot.Model, c *boot.Config, insight *insight.Insight) {
	for _, insight := range *insight {
		if err := m.InsertInspectionNtpInfo(&model.NtpInfo{
			InspectionId: c.InspectionId,
			NodeIp:       insight.NodeIp,
			Offset:       insight.Ntp.Offset,
		}); err != nil {
			log.Error("insert ntp info:", err)
			return
		}
	}
}
