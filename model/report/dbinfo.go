package report

type DBInfo struct {
	InspectionId string `json:"-"`
	DB           string `json:"schema"`
	Table        string `json:"table"`
	Index        int    `json:"index"`
}

func (m *report) GetInspectionDBInfo(inspectionId string) ([]*DBInfo, error) {
	infos := []*DBInfo{}

	if err := m.db.Where(&DBInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) GetTablesWithoutIndex(inspectionId string) ([]*DBInfo, error) {
	infos := []*DBInfo{}

	if err := m.db.Where(&DBInfo{InspectionId: inspectionId, Index: 0}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionDBInfo(inspectionId string) error {
	return m.db.Delete(&DBInfo{InspectionId: inspectionId}).Error()
}

func (m *report) InsertInspectionDBInfo(info *DBInfo) error {
	return m.db.Create(info).Error()
}
