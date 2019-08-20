package report

type DmesgLog struct {
	InspectionId string `json:"-"`
	NodeIp       string `json:"node_ip"`
	Log          string `json:"log"`
}

func (m *report) GetInspectionDmesg(inspectionId string) ([]*DmesgLog, error) {
	logs := []*DmesgLog{}

	if err := m.db.Where(&DmesgLog{InspectionId: inspectionId}).Find(&logs).Error(); err != nil {
		return nil, err
	}

	return logs, nil
}

func (m *report) ClearInspectionDmesgLog(inspectionId string) error {
	return m.db.Delete(&DmesgLog{}, "inspection_id = ?", inspectionId).Error()
}

func (m *report) InsertInspectionDmesgLog(info *DmesgLog) error {
	return m.db.Create(info).Error()
}
