package resource

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type analyzeTask struct{}

func Analyze() *analyzeTask {
	return &analyzeTask{}
}

// Check if the avg resource useage exceed 20%
func (t *analyzeTask) Run(m *boot.Model, c *boot.Config) {
	resources, err := m.GetInspectionResourceInfo(c.InspectionId)
	if err != nil {
		log.Error("get resource info:", err)
		return
	}

	for _, res := range resources {
		desc := "please increase resources appropriately"
		if res.Max.GetTag("status") != "" {
			m.InsertSymptom(res.Max.GetTag("status"), res.Max.GetTag("message"), desc)
		}
		if res.Avg.GetTag("status") != "" {
			m.InsertSymptom(res.Avg.GetTag("status"), res.Avg.GetTag("message"), desc)
		}
	}
}
