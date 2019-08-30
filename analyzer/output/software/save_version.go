package software

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/insight"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type saveSoftwareVersionTask struct{}

func SaveSoftwareVersion() *saveSoftwareVersionTask {
	return &saveSoftwareVersionTask{}
}

// Save each component's version to database
func (t *saveSoftwareVersionTask) Run(c *boot.Config, m *boot.Model, insights *insight.Insight) {
	vm := make(map[string][]*model.SoftwareInfo)
	for _, insight := range *insights {
		versions := loadSoftwareVersion(insight)
		for _, v := range versions {
			vm[v.component] = append(vm[v.component], &model.SoftwareInfo{
				InspectionId: c.InspectionId,
				NodeIp:       v.ip,
				Component:    v.component,
				Version:      utils.NewTagdString(v.version, nil),
			})
		}
	}

	for _, vs := range vm {
		for idx, v := range vs {
			if idx != 0 && v.Version.GetValue() != vs[idx-1].Version.GetValue() {
				v.Version.SetTag("status", "error")
				v.Version.SetTag("message", fmt.Sprintf("version of %s on node %s not same with other node", v.Component, v.NodeIp))
			}
			if err := m.InsertInspectionSoftwareInfo(v); err != nil {
				log.Error("insert inspection component version:", err)
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
