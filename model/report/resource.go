package report

type ResourceInfo struct {
	InspectionId string
	Name         string
	Duration     string
	Value        float64
	Status       string
}

func (m *report) GetInspectionResourceInfo(inspectionId string) ([]*ResourceInfo, error) {
	infos := []*ResourceInfo{}

	if err := m.db.Where(&ResourceInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionResourceInfo(inspectionId string) error {
	return m.db.Delete(&ResourceInfo{InspectionId: inspectionId}).Error()
}

func (m *report) InsertInspectionResourceInfo(info *ResourceInfo) error {
	return m.db.Create(info).Error()
}