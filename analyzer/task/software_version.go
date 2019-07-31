package task

import (
	log "github.com/sirupsen/logrus"
)

type SaveSoftwareVersionTask struct {
	BaseTask
}

func SaveSoftwareVersion(base BaseTask) Task {
	return &SaveSoftwareVersionTask{base}
}

func (t *SaveSoftwareVersionTask) Run() error {
	insights := t.data.insight
	for _, insight := range insights {
		versions := loadSoftwareVersion(insight)
		for _, v := range versions {
			if _, err := t.db.Exec(
				`INSERT INTO software_version(inspection, node_ip, component, version) VALUES(?, ?, ?, ?)`,
				t.inspectionId, v.ip, v.component, v.version,
			); err != nil {
				log.Error("db.Exec: ", err)
				return err
			}
		}
	}
	return nil
}

type SoftwareVersion struct {
	ip        string
	component string
	version   string
}

func loadSoftwareVersion(insight *InsightInfo) []SoftwareVersion {
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
