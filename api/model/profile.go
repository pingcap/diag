package model

import (
	log "github.com/sirupsen/logrus"
)

type Profile struct {
	Uuid string `json:"uuid"`
}

func (m *Model) ListProfiles(instanceId string) ([]*Profile, error) {
	profiles := []*Profile{}

	rows, err := m.db.Query(
		"SELECT distinct(inspection) FROM inspection_items WHERE name = 'profile' AND status <> 'none'",
	)
	if err != nil {
		log.Error("failed to call db.Query:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		profile := Profile{}

		err := rows.Scan(
			&profile.Uuid,
		)
		if err != nil {
			log.Error("db.Query:", err)
			return nil, err
		}

		profiles = append(profiles, &profile)
	}

	return profiles, nil
}
