package software

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type analyzeVersionTask struct{}

func AnalyzeVersion() *analyzeVersionTask {
	return &analyzeVersionTask{}
}

// Check if all software versions are same
func (t *analyzeVersionTask) Run(m *boot.Model, c *boot.Config) {
	comps, err := m.GetInspectionSoftwareInfo(c.InspectionId)
	if err != nil {
		log.Error("get component version:", err)
		m.InsertSymptom("exception", "error on get component version", "contact foresight developer")
		return
	}

	for _, comp := range comps {
		if comp.Version.GetTag("status") != "" {
			msg := comp.Version.GetTag("message")
			desc := fmt.Sprintf("make sure all %s have the correct", comp)
			m.InsertSymptom("error", msg, desc)
		}
	}
}
