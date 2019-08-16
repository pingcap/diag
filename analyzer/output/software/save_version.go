package software

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/insight"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type saveSoftwareVersionTask struct{}

func SaveSoftwareVersion() *saveSoftwareVersionTask {
	return &saveSoftwareVersionTask{}
}

// Save each component's version to database
func (t *saveSoftwareVersionTask) Run(c *boot.Config, m *boot.Model, insights *insight.Insight) {
	for _, insight := range *insights {
		versions := loadSoftwareVersion(insight)
		for _, v := range versions {
			if err := m.InsertInspectionSoftwareInfo(&model.SoftwareInfo{
				InspectionId: c.InspectionId,
				NodeIp:       v.ip,
				Component:    v.component,
				Version:      v.version,
			}); err != nil {
				log.Error("insert inspection component version:", err)
				return
			}
		}
	}
}

func loadSoftwareVersion(insight *insight.InsightInfo) []SoftwareVersion {
	var versions []SoftwareVersion
	ip := insight.NodeIp
	for _, item := range insight.Meta.Tidb {
		version := SoftwareVersion{
			ip:        ip,
			component: "tidb",
			version:   item.Version,
		}
		versions = append(versions, version)
	}
	for _, item := range insight.Meta.Tikv {
		version := SoftwareVersion{
			ip:        ip,
			component: "tikv",
			version:   item.Version,
		}
		versions = append(versions, version)
	}
	for _, item := range insight.Meta.Pd {
		version := SoftwareVersion{
			ip:        ip,
			component: "pd",
			version:   item.Version,
		}
		versions = append(versions, version)
	}
	return versions
}
