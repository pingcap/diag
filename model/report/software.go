package report

type SoftwareInfo struct {
	InspectionId string
	NodeIp       string `json:"node_ip"`
	Component    string `json:"component"`
	Version      string `json:"version"`
}

func (m *report) GetInspectionSoftwareInfo(inspectionId string) ([]*SoftwareInfo, error) {
	infos := []*SoftwareInfo{}

	if err := m.db.Where(&SoftwareInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionSoftwareInfo(inspectionId string) error {
	return m.db.Delete(&SoftwareInfo{InspectionId: inspectionId}).Error()
}

func (m *report) InsertInspectionSoftwareInfo(info *SoftwareInfo) error {
	return m.db.Create(info).Error()
}
