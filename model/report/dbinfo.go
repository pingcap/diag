package report

import (
	ti "github.com/pingcap/tidb-foresight/utils/tagd-value/int64"
)

type DBInfo struct {
	InspectionId string   `json:"-"`
	DB           string   `json:"schema"`
	Table        string   `json:"table"`
	Index        ti.Int64 `json:"index"`
}

func (m *report) GetInspectionDBInfo(inspectionId string) ([]*DBInfo, error) {
	infos := []*DBInfo{}

	if err := m.db.Where(&DBInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionDBInfo(inspectionId string) error {
	return m.db.Delete(&DBInfo{}, "inspection_id = ?", inspectionId).Error()
}

func (m *report) InsertInspectionDBInfo(info *DBInfo) error {
	return m.db.Create(info).Error()
}
