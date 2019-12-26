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

type checkStruct struct {
	Ip     string
	Comp   string
	Config string
	Port   string
}

// Check if all software configs are same
func (t *analyzeConfigTask) Run(m *boot.Model, c *boot.Config) {
	t.m = m

	configs, err := m.GetInspectionConfigInfo(c.InspectionId)
	if err != nil {
		log.Error("get component config:", err)
		m.InsertSymptom("exception", "error on get component config", "contact foresight developer")
	}
	configMap := make(map[string][]checkStruct)
	for _, config := range configs {
		configMap[config.Component] = append(configMap[config.Component], checkStruct{
			Ip:     config.NodeIp,
			Comp:   config.Component,
			Config: config.Config,
			Port:   config.Port,
		})
	}

	for k, v := range configMap {
		t.checkComponentConfig(k, v)
	}
}

func (t *analyzeConfigTask) checkComponentConfig(comp string, configs []checkStruct) {
	for index, c1 := range configs {
		for _, c2 := range configs[:index] {
			if c1.Config != c2.Config {
				msg := fmt.Sprintf("%s config is not identical", comp)
				desc := fmt.Sprintf("make sure all %s have the same config (%v:%v has config %v, but %v:%v has config %v)",
					comp, c1.Ip, c1.Port, c1.Config, c2.Ip, c2.Port, c2.Config)
				t.m.InsertSymptom("error", msg, desc)
				return
			}
		}
	}
}
