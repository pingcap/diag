package report

type Symptom struct {
	InspectionId string
	Status       string `json:"status"`
	Message      string `json:"message"`
	Description  string `json:"description"`
}

func (m *report) GetInspectionSymptoms(inspectionId string) ([]*Symptom, error) {
	infos := []*Symptom{}

	if err := m.db.Where(&Symptom{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionSymptom(inspectionId string) error {
	return m.db.Delete(&Symptom{InspectionId: inspectionId}).Error()
}

func (m *report) InsertInspectionSymptom(symptom *Symptom) error {
	return m.db.Create(symptom).Error()
}
