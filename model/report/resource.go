package report

import (
	ts "github.com/pingcap/tidb-foresight/utils/tagd-value/string"
)

type ResourceInfo struct {
	InspectionId string    `json:"-"`
	Name         string    `json:"name"`
	Duration     string    `json:"duration"`
	Avg          ts.String `json:"avg"`
	Max          ts.String `json:"max"`
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
