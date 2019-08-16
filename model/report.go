package model

import (
	"github.com/pingcap/tidb-foresight/model/report"
)

func (m *Model) GetInspectionBasicInfo(inspectionId string) (*report.BasicInfo, error) {
	return report.GetBasicInfo(m.db, inspectionId)
}

func (m *Model) GetInspectionAlertInfo(inspectionId string) ([]*report.AlertInfo, error) {
	return report.GetAlertInfo(m.db, inspectionId)
}

func (m *Model) GetInspectionConfigInfo(inspectionId string) ([]*report.ConfigInfo, error) {
	return report.GetConfigInfo(m.db, inspectionId)
}

func (m *Model) GetInspectionDBInfo(inspectionId string) ([]*report.DBInfo, error) {
	return report.GetDBInfo(m.db, inspectionId)
}

func (m *Model) GetInspectionDmesg(inspectionId string) ([]*report.DmesgLog, error) {
	return report.GetDmesgLog(m.db, inspectionId)
}

func (m *Model) GetInspectionHardwareInfo(inspectionId string) ([]*report.HardwareInfo, error) {
	return report.GetHardwareInfo(m.db, inspectionId)
}

func (m *Model) GetInspectionNetworkInfo(inspectionId string) ([]*report.NetworkInfo, error) {
	return report.GetNetworkInfo(m.db, inspectionId)
}

func (m *Model) GetInspectionResourceInfo(inspectionId string) ([]*report.ResourceInfo, error) {
	return report.GetResourceInfo(m.db, inspectionId)
}

func (m *Model) GetInspectionSlowLog(inspectionId string) ([]*report.SlowLogInfo, error) {
	return report.GetSlowLogInfo(m.db, inspectionId)
}

func (m *Model) GetInspectionSoftwareInfo(inspectionId string) ([]*report.SoftwareInfo, error) {
	return report.GetSoftwareInfo(m.db, inspectionId)
}

func (m *Model) GetInspectionSymptoms(inspectionId string) ([]*report.Symptom, error) {
	return report.GetSymptomInfo(m.db, inspectionId)
}
