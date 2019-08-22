package resource

import (
	"fmt"

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
		if res.Value.GetTag("status") == "abnormal" {
			msg := fmt.Sprintf("%s Resource utilization/%s too high", res.Name, res.Duration)
			defer m.InsertSymptom(
				"error",
				msg,
				"please increase resources appropriately",
			)
		}
	}
}
