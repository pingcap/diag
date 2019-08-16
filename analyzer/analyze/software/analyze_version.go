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
	versions, err := m.GetInspectionSoftwareInfo(c.InspectionId)
	if err != nil {
		log.Error("get component version:", err)
		m.InsertSymptom("exception", "error on get component version", "contact foresight developer")
		return
	}
	versionMap := make(map[string][]string)
	for _, version := range versions {
		versionMap[version.Component] = append(versionMap[version.Component], version.Version)
	}

	for k, v := range versionMap {
		t.checkComponentVersion(m, k, v)
	}
}

func (t *analyzeVersionTask) checkComponentVersion(m *boot.Model, comp string, versions []string) {
	for _, v1 := range versions {
		for _, v2 := range versions {
			if v1 != v2 {
				msg := fmt.Sprintf("%s version is not identical", comp)
				desc := fmt.Sprintf("make sure all %s have the same version", comp)
				m.InsertSymptom("error", msg, desc)
				return
			}
		}
	}
}
