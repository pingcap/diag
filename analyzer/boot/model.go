package boot

import (
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

// model with tool method
type Model struct {
	inspectionId string
	model.Model
}

func NewModel(inspectionId string, m model.Model) *Model {
	return &Model{inspectionId, m}
}

func (m *Model) InsertSymptom(status, message, description string) {
	if err := m.InsertInspectionSymptom(&model.Symptom{
		InspectionId: m.inspectionId,
		Status:       status,
		Message:      message,
		Description:  description,
	}); err != nil {
		log.Panic("insert symptom:", err)
	}
}
