package report

import (
	"time"
)

type AlertInfo struct {
	InspectionId string    `json:"-"`
	Name         string    `json:"name"`
	Value        string    `json:"value"`
	Time         time.Time `json:"time"`
}

func (m *report) GetInspectionAlertInfo(inspectionId string) ([]*AlertInfo, error) {
	infos := []*AlertInfo{}

	if err := m.db.Where(&AlertInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionAlertInfo(inspectionId string) error {
	return m.db.Delete(&AlertInfo{InspectionId: inspectionId}).Error()
}

func (m *report) InsertInspectionAlertInfo(info *AlertInfo) error {
	return m.db.Create(info).Error()
}
