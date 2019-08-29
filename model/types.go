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

type Symptom = report.Symptom
