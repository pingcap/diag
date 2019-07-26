package report

import (
	log "github.com/sirupsen/logrus"
)

type Symptom struct {
	Type		string	`json:"type"`
	Description string  `json:"description"`
	Suggestion  string  `json:"suggesion"`
}

func (r *Report) loadSymptoms() error {
	symptoms := []*Symptom{}

	rows, err := r.db.Query(
		`SELECT type, description, suggestion FROM inspection_symptoms WHERE inspection = ?`,
		r.inspectionId,
	)
	if err != nil {
		log.Error("db.Query:", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		symptom := Symptom{}
		err = rows.Scan(&symptom.Type, &symptom.Description, &symptom.Suggestion)
		if err != nil {
			log.Error("db.Query:", err)
			return err
		}
		symptoms = append(symptoms, &symptom)
	}

	r.Symptoms = symptoms
	return nil
}
