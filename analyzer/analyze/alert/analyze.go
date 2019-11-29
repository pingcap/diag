package alert

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type analyzeTask struct{}

func Analyze() *analyzeTask {
	return &analyzeTask{}
}

// Check if there is any alert from prometheus
func (t *analyzeTask) Run(m *boot.Model, c *boot.Config) {
	m.UpdateInspectionMessage(c.InspectionId, "analyzing alert info...")
	if alerts, err := m.GetInspectionAlertInfo(c.InspectionId); err != nil {
		log.Error("count alert info:", err)
		return
	} else if len(alerts) > 0 {
		msg := "there are alert information in the cluster"
		desc := "please check the alert information"
		m.InsertSymptom("warning", msg, desc)
	}
}
