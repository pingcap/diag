package report

type ConfigInfo struct {
	InspectionId string `json:"-"`
	NodeIp       string `json:"node_ip"`
	Port         string `json:"port"`
	Component    string `json:"component"`
	Config       string `json:"config"`
	// TODO: filling the open file limit here.
	OpenFileLimit int `json:"open_file_limit"`
}

func (m *report) GetInspectionConfigInfo(inspectionId string) ([]*ConfigInfo, error) {
	infos := []*ConfigInfo{}

	if err := m.db.Where(&ConfigInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionConfigInfo(inspectionId string) error {
	return m.db.Delete(&ConfigInfo{}, "inspection_id = ?", inspectionId).Error()
}

func (m *report) InsertInspectionConfigInfo(info *ConfigInfo) error {
	return m.db.Create(info).Error()
}
