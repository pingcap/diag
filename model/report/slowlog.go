package report

import (
	"time"
)

type SlowLogInfo struct {
	InspectionId string    `json:"-"`
	Time         time.Time `json:"time"`
	Query        string    `json:"query"`
}

func (m *report) GetInspectionSlowLog(inspectionId string) ([]*SlowLogInfo, error) {
	infos := []*SlowLogInfo{}

	if err := m.db.Where(&SlowLogInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionSlowLog(inspectionId string) error {
	return m.db.Delete(&SlowLogInfo{InspectionId: inspectionId}).Error()
}

func (m *report) InsertInspectionSlowLog(info *SlowLogInfo) error {
	return m.db.Create(info).Error()
}
