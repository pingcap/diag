package report

type NetworkInfo struct {
	InspectionId string
	NodeIp       string `json:"node_ip"`
	Connections  int64  `json:"connections"`
	Recv         int64  `json:"recv"`
	Send         int64  `json:"send"`
	BadSeg       int64  `json:"bad_seg"`
	Retrans      int64  `json:"retrans"`
}

func (m *report) GetInspectionNetworkInfo(inspectionId string) ([]*NetworkInfo, error) {
	infos := []*NetworkInfo{}

	if err := m.db.Where(&NetworkInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionNetworkInfo(inspectionId string) error {
	return m.db.Delete(&NetworkInfo{InspectionId: inspectionId}).Error()
}

func (m *report) InsertInspectionNetworkInfo(info *NetworkInfo) error {
	return m.db.Create(info).Error()
}
