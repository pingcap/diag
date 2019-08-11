package software

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/insight"
	log "github.com/sirupsen/logrus"
)

type saveSoftwareVersionTask struct{}

func SaveSoftwareVersion() *saveSoftwareVersionTask {
	return &saveSoftwareVersionTask{}
}

// Save each component's version to database
func (t *saveSoftwareVersionTask) Run(c *boot.Config, db *boot.DB, insights *insight.Insight) {
	for _, insight := range *insights {
		versions := loadSoftwareVersion(insight)
		for _, v := range versions {
			if _, err := db.Exec(
				`INSERT INTO software_version(inspection, node_ip, component, version) VALUES(?, ?, ?, ?)`,
				c.InspectionId, v.ip, v.component, v.version,
			); err != nil {
				log.Error("db.Exec:", err)
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
