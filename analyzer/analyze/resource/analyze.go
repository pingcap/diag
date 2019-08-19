package resource

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

const THRESHOLD = 20

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
		if res.Value > THRESHOLD {
			msg := fmt.Sprintf("%s Resource utilization/%s exceeds %d%%", res.Name, res.Duration, THRESHOLD)
			defer m.InsertSymptom(
				"error",
				msg,
				"please increase resources appropriately",
			)
		}
	}
}
