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

	// vm is a map for
	// <component, <ip, array of version>>
	// and version is an SoftwareVersion object.
	vm := make(map[string]map[string][]SoftwareVersion)
	for _, v := range versions {
		if vm[v.component] == nil {
			vm[v.component] = make(map[string][]SoftwareVersion)
		}
		vm[v.component][v.ip] = append(vm[v.component][v.ip], v)
	}

	loadVersionString := func(versions []SoftwareVersion) []string {
		retList := make([]string, len(versions), len(versions))
		for i, version := range versions {
			retList[i] = version.version
		}
		return retList
	}

	loadOSString := func(versions []SoftwareVersion) []string {
		retList := make([]string, 0)
		for _, version := range versions {
			if version.os != "" {
				retList = append(retList, version.os)
			}
		}
		return retList
	}

	loadFSString := func(versions []SoftwareVersion) []string {
		retList := make([]string, 0)
		for _, version := range versions {
			if version.fs != "" {
				retList = append(retList, version.fs)
			}
		}
		return retList
	}

	loadNetworkString := func(versions []SoftwareVersion) []string {
		retList := make([]string, 0)
		for _, version := range versions {
			if version.network != "" {
				retList = append(retList, version.network)
			}
		}
		return retList
	}

	// comp is a string represents the component. eg: tidb.
	// hm is an map like <ip, array of versions>.
	for comp, hm := range vm {
		versions := make([]*model.SoftwareInfo, 0)
		for ip, vs := range hm {
			vss := loadVersionString(vs)
			oss := loadOSString(vs)
			fss := loadFSString(vs)
			networks := loadNetworkString(vs)

			v := ts.New(strings.Join(vss, ","), nil)
			if !identity(vss) {
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
				// TODO: fill the message below
				OS:           strings.Join(oss, ","),
				FileSystem:   strings.Join(fss, ","),
				NetworkDrive: strings.Join(networks, ","),
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
	// load all fs and network drives in the system
	// TODO: finish the message below and make sure which fs or network drive is used by the process.
	fsList := make([]string, 0)
	networkDriveList := make([]string, 0)

	for _, network := range insight.Sysinfo.Network {
		if network.Driver != nil {
			networkDriveList = append(networkDriveList, *network.Driver)
		}
	}

	for _, partions := range insight.Partitions {
		for _, dev := range partions.Subdev {
			if dev.Mount != nil && dev.Mount.FileSystem != nil {
				fsList = append(fsList, *dev.Mount.FileSystem)
			}
		}
	}

	for _, item := range insight.Meta.Tidb {
		version := SoftwareVersion{
			ip:        ip,
			component: "tidb",
			version:   item.Version,

			os:      insight.Sysinfo.Os.Name,
			fs:      strings.Join(fsList, ","),
			network: strings.Join(networkDriveList, ","),
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
