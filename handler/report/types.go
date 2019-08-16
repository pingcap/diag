package report

import (
	"github.com/pingcap/tidb-foresight/model/report"
)

type SymptomGeter interface {
	GetInspectionSymptoms(inspectionId string) ([]*report.Symptom, error)
}

type SlowLogGeter interface {
	GetInspectionSlowLog(inspectionId string) ([]*report.SlowLogInfo, error)
}

type BasicInfoGeter interface {
	GetInspectionBasicInfo(inspectionId string) (*report.BasicInfo, error)
}

type AlertInfoGeter interface {
	GetInspectionAlertInfo(inspectionId string) ([]*report.AlertInfo, error)
}

type ConfigInfoGeter interface {
	GetInspectionConfigInfo(inspectionId string) ([]*report.ConfigInfo, error)
}

type DBInfoGeter interface {
	GetInspectionDBInfo(inspectionId string) ([]*report.DBInfo, error)
}

type DmesgGeter interface {
	GetInspectionDmesg(inspectionId string) ([]*report.DmesgLog, error)
}

type HardwareInfoGeter interface {
	GetInspectionHardwareInfo(inspectionId string) ([]*report.HardwareInfo, error)
}

type NetworkInfoGeter interface {
	GetInspectionNetworkInfo(inspectionId string) ([]*report.NetworkInfo, error)
}

type ResourceInfoGeter interface {
	GetInspectionResourceInfo(inspectionId string) ([]*report.ResourceInfo, error)
}

type SoftwareInfoGeter interface {
	GetInspectionSoftwareInfo(inspectionId string) ([]*report.SoftwareInfo, error)
}
