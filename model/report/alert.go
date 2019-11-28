package report

import (
	"time"

	ts "github.com/pingcap/tidb-foresight/utils/tagd-value/string"
)

type AlertInfo struct {
	InspectionId string    `json:"-"`
	Name         string    `json:"name"`
	Value        ts.String `json:"value"`
	Time         time.Time `json:"time"`
	Description  string    `json:"description"`
}

func (m *report) GetInspectionAlertInfo(inspectionId string) ([]*AlertInfo, error) {
	infos := []*AlertInfo{}

	if err := m.db.Where(&AlertInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionAlertInfo(inspectionId string) error {
	return m.db.Delete(&AlertInfo{}, "inspection_id = ?", inspectionId).Error()
}

func (m *report) InsertInspectionAlertInfo(info *AlertInfo) error {
	return m.db.Create(info).Error()
}
