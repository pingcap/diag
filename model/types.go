package model

import (
	"github.com/pingcap/tidb-foresight/model/config"
	"github.com/pingcap/tidb-foresight/model/inspection"
	"github.com/pingcap/tidb-foresight/model/instance"
	"github.com/pingcap/tidb-foresight/model/logs"
	"github.com/pingcap/tidb-foresight/model/profile"
	"github.com/pingcap/tidb-foresight/model/report"
)

type Instance = instance.Instance

type Component struct {
	Name string `json:"name"`
	Ip   string `json:"ip"`
	Port string `json:"port"`
}

// The topology.json presentation
type Topology struct {
	// cluster name of this inspection
	ClusterName string `json:"cluster_name"`
	// cluster version from inventory.ini
	ClusterVersion string `json:"tidb_version"`
	// the hosts of the cluster
	Hosts []struct {
		Ip         string `json:"ip"`
		Components []struct {
			// the name of compoennt, eg. tidb, tikv, pd
			Name string `json:"name"`
			// where the component deployed on remote host
			DeployDir string `json:"deploy_dir"`
			// the port this component listen on
			Port string `json:"port"`
			// status port (only tidb)
			StatusPort string `json:"status_port"`
			// if this component alive
			Status string `json:"-"`
		} `json:"components"`
	} `json:"hosts"`
}

type Config = config.Config

type Inspection = inspection.Inspection

type Profile = profile.Profile

type LogEntity = logs.LogEntity

type BasicInfo = report.BasicInfo

type AlertInfo = report.AlertInfo

type ConfigInfo = report.ConfigInfo

type Item = report.Item

type DBInfo = report.DBInfo

type DmesgLog = report.DmesgLog

type HardwareInfo = report.HardwareInfo

type NetworkInfo = report.NetworkInfo

type ResourceInfo = report.ResourceInfo

type SlowLogInfo = report.SlowLogInfo

type SoftwareInfo = report.SoftwareInfo

type NtpInfo = report.NtpInfo

type TopologyInfo = report.TopologyInfo

type Symptom = report.Symptom
