package report

import (
	"github.com/pingcap/tidb-foresight/utils"
)

type ResourceInfo struct {
	InspectionId string
	Name         string
	Duration     string
	Avg          utils.TagdFloat64
	Max          utils.TagdFloat64
}

func (m *report) GetInspectionResourceInfo(inspectionId string) ([]*ResourceInfo, error) {
	infos := []*ResourceInfo{}

	if err := m.db.Where(&ResourceInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionResourceInfo(inspectionId string) error {
	return m.db.Delete(&ResourceInfo{}, "inspection_id = ?", inspectionId).Error()
}

func (m *report) InsertInspectionResourceInfo(info *ResourceInfo) error {
	return m.db.Create(info).Error()
}
