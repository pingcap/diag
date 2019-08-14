package report

import (
	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

type Symptom struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Description string `json:"description"`
}

// deprecated
func (r *Report) loadSymptoms() error {
	symptoms := []*Symptom{}

	rows, err := r.db.Query(
		`SELECT status, message, description FROM inspection_symptoms WHERE inspection = ?`,
		r.inspectionId,
	)
	if err != nil {
		log.Error("db.Query:", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		symptom := Symptom{}
		err = rows.Scan(&symptom.Status, &symptom.Message, &symptom.Description)
		if err != nil {
			log.Error("db.Query:", err)
			return err
		}
		symptoms = append(symptoms, &symptom)
	}

	r.Symptoms = symptoms
	return nil
}

func GetSymptomInfo(db db.DB, inspectionId string) ([]*Symptom, error) {
	symptoms := []*Symptom{}

	rows, err := db.Query(
		`SELECT status, message, description FROM inspection_symptoms WHERE inspection = ?`,
		inspectionId,
	)
	if err != nil {
		log.Error("db.Query:", err)
		return symptoms, err
	}
	defer rows.Close()

	for rows.Next() {
		symptom := Symptom{}
		err = rows.Scan(&symptom.Status, &symptom.Message, &symptom.Description)
		if err != nil {
			log.Error("db.Query:", err)
			return symptoms, err
		}
		symptoms = append(symptoms, &symptom)
	}

	return symptoms, nil
}
