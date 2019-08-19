package software

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type analyzeConfigTask struct {
	m *boot.Model
}

func AnalyzeConfig() *analyzeConfigTask {
	return &analyzeConfigTask{}
}

// Check if all software configs are same
func (t *analyzeConfigTask) Run(m *boot.Model, c *boot.Config) {
	t.m = m

	configs, err := m.GetInspectionConfigInfo(c.InspectionId)
	if err != nil {
		log.Error("get component config:", err)
		m.InsertSymptom("exception", "error on get component config", "contact foresight developer")
	}
	configMap := make(map[string][]string)
	for _, config := range configs {
		configMap[config.Component] = append(configMap[config.Component], config.Config)
	}

	for k, v := range configMap {
		t.checkComponentConfig(k, v)
	}
}

func (t *analyzeConfigTask) checkComponentConfig(comp string, configs []string) {
	for _, c1 := range configs {
		for _, c2 := range configs {
			if c1 != c2 {
				msg := fmt.Sprintf("%s config is not identical", comp)
				desc := fmt.Sprintf("make sure all %s have the same config", comp)
				t.m.InsertSymptom("error", msg, desc)
				return
			}
		}
	}
}
