package report

import (
	"github.com/pingcap/tidb-foresight/utils"
)

type NtpInfo struct {
	InspectionId string
	NodeIp       string
	Offset       utils.TagdFloat64
}

func (m *report) GetInspectionNtpInfo(inspectionId string) ([]*NtpInfo, error) {
	infos := []*NtpInfo{}

	if err := m.db.Where(&NtpInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionNtpInfo(inspectionId string) error {
	return m.db.Delete(&NtpInfo{}, "inspection_id = ?", inspectionId).Error()
}

func (m *report) InsertInspectionNtpInfo(info *NtpInfo) error {
	return m.db.Create(info).Error()
}
