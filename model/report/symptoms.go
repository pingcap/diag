package report

import (
	log "github.com/sirupsen/logrus"
)

type Symptom struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Description string `json:"description"`
}

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
