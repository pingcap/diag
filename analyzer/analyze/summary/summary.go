package summary

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type summaryTask struct{}

func Summary() *summaryTask {
	return &summaryTask{}
}

// Change the inspection status to success
func (t *summaryTask) Run(m *boot.Model, c *boot.Config) {
	if err := m.UpdateInspectionStatus(c.InspectionId, "success"); err != nil {
		log.Error("update inspection status:", err)
		return
	}
}
