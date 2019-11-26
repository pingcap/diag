package report

import (
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wrapper/db"
)

type Model interface {
	GetInspectionBasicInfo(inspectionId string) (*BasicInfo, error)
	GetInspectionAlertInfo(inspectionId string) ([]*AlertInfo, error)
	GetInspectionConfigInfo(inspectionId string) ([]*ConfigInfo, error)
	GetInspectionDmesg(inspectionId string) ([]*DmesgLog, error)
	GetInspectionHardwareInfo(inspectionId string) ([]*HardwareInfo, error)
	GetInspectionNetworkInfo(inspectionId string) ([]*NetworkInfo, error)
	GetInspectionResourceInfo(inspectionId string) ([]*ResourceInfo, error)
	GetInspectionSlowLog(inspectionId string) ([]*SlowLogInfo, error)
	GetInspectionSoftwareInfo(inspectionId string) ([]*SoftwareInfo, error)
	GetInspectionSymptoms(inspectionId string) ([]*Symptom, error)
	GetInspectionDBInfo(inspectionId string) ([]*DBInfo, error)
	GetInspectionNtpInfo(inspectionId string) ([]*NtpInfo, error)
	GetInspectionTopologyInfo(inspectionId string) ([]*TopologyInfo, error)
	ClearInspectionItem(inspectionId string) error
	ClearInspectionSymptom(inspectionId string) error
	ClearInspectionBasicInfo(inspectionId string) error
	ClearInspectionDBInfo(inspectionId string) error
	ClearInspectionSlowLog(inspectionId string) error
	ClearInspectionNetworkInfo(inspectionId string) error
	ClearInspectionAlertInfo(inspectionId string) error
	ClearInspectionHardwareInfo(inspectionId string) error
	ClearInspectionDmesgLog(inspectionId string) error
	ClearInspectionSoftwareInfo(inspectionId string) error
	ClearInspectionConfigInfo(inspectionId string) error
	ClearInspectionResourceInfo(inspectionId string) error
	ClearInspectionTopologyInfo(inspectionId string) error
	ClearInspectionNtpInfo(inspectionId string) error
	InsertInspectionItem(*Item) error
	InsertInspectionSymptom(*Symptom) error
	InsertInspectionBasicInfo(*BasicInfo) error
	InsertInspectionDBInfo(*DBInfo) error
	InsertInspectionSlowLog(*SlowLogInfo) error
	InsertInspectionNetworkInfo(*NetworkInfo) error
	InsertInspectionAlertInfo(*AlertInfo) error
	InsertInspectionHardwareInfo(*HardwareInfo) error
	InsertInspectionDmesgLog(*DmesgLog) error
	InsertInspectionSoftwareInfo(*SoftwareInfo) error
	InsertInspectionConfigInfo(*ConfigInfo) error
	InsertInspectionResourceInfo(*ResourceInfo) error
	InsertInspectionTopologyInfo(*TopologyInfo) error
	InsertInspectionNtpInfo(info *NtpInfo) error
}

func New(db db.DB) Model {
	utils.MustInitSchema(
		db,
		&Item{},
		&DBInfo{},
		&AlertInfo{},
		&BasicInfo{},
		&ConfigInfo{},
		&DmesgLog{},
		&HardwareInfo{},
		&NetworkInfo{},
		&ResourceInfo{},
		&SlowLogInfo{},
		&NtpInfo{},
		&SoftwareInfo{},
		&TopologyInfo{},
		&Symptom{},
	)
  
	db.Debug().AutoMigrate(&SoftwareInfo{}, &ConfigInfo{})
	return &report{db}
}

type report struct {
	db db.DB
}
