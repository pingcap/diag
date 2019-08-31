package software

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/insight"
	"github.com/pingcap/tidb-foresight/model"
	ts "github.com/pingcap/tidb-foresight/utils/tagd-value/string"
	log "github.com/sirupsen/logrus"
)

type saveSoftwareVersionTask struct{}

func SaveSoftwareVersion() *saveSoftwareVersionTask {
	return &saveSoftwareVersionTask{}
}

// Save each component's version to database
func (t *saveSoftwareVersionTask) Run(c *boot.Config, m *boot.Model, insights *insight.Insight) {
	versions := []SoftwareVersion{}
	for _, insight := range *insights {
		versions = append(versions, loadSoftwareVersion(insight)...)
	}

	vm := make(map[string]map[string][]string)
	for _, v := range versions {
		if vm[v.component] == nil {
			vm[v.component] = make(map[string][]string)
		}
		vm[v.component][v.ip] = append(vm[v.component][v.ip], v.version)
	}

	for comp, hm := range vm {
		versions := make([]*model.SoftwareInfo, 0)
		for ip, vs := range hm {
			v := ts.New(strings.Join(vs, ","), nil)
			if !identity(vs) {
				msg := fmt.Sprintf(
					"it seems you have multiple version of %s on %s, foresight can't decide which one is correct, please confirm it yourself.",
					comp, ip,
				)
				// TODO: use warning
				v.SetTag("status", "error")
				v.SetTag("message", msg)
			}
			versions = append(versions, &model.SoftwareInfo{
				InspectionId: c.InspectionId,
				NodeIp:       ip,
				Component:    comp,
				Version:      v,
			})
		}
		sort.Slice(versions, func(i, j int) bool {
			return len(strings.Split(versions[i].Version.GetValue(), ",")) < len(strings.Split(versions[j].Version.GetValue(), ","))
		})
		for idx, v := range versions {
			if idx == 0 {
				if strings.Contains(v.Version.GetValue(), ",") {
					break
				}
				continue
			}
			vs := strings.Split(v.Version.GetValue(), ",")
			if !contains(vs, versions[0].Version.GetValue()) {
				msg := fmt.Sprintf(
					"we think the version of %s on node %s should be %s, but get %s",
					v.Component,
					v.NodeIp,
					versions[0].Version.GetValue(),
					v.Version.GetValue(),
				)
				v.Version.SetTag("status", "error")
				v.Version.SetTag("message", msg)
			}
		}
		for _, v := range versions {
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

func identity(ss []string) bool {
	if len(ss) < 2 {
		return true
	}

	for idx, s := range ss {
		if idx != 0 && s != ss[0] {
			return false
		}
	}

	return true
}

func contains(ss []string, s string) bool {
	for _, str := range ss {
		if str == s {
			return true
		}
	}
	return false
}
