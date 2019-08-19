package report

type HardwareInfo struct {
	InspectionId string `json:"-"`
	NodeIp       string `json:"node_ip"`
	Cpu          string `json:"cpu"`
	Memory       string `json:"memory"`
	Disk         string `json:"disk"`
	Network      string `json:"network"`
}

func (m *report) GetInspectionHardwareInfo(inspectionId string) ([]*HardwareInfo, error) {
	infos := []*HardwareInfo{}

	if err := m.db.Where(&HardwareInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionHardwareInfo(inspectionId string) error {
	return m.db.Delete(&HardwareInfo{InspectionId: inspectionId}).Error()
}

func (m *report) InsertInspectionHardwareInfo(info *HardwareInfo) error {
	return m.db.Create(info).Error()
}
