package model

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

type LogEntity struct {
	Id           string `json:"uuid"`
	InstanceName string `json:"instance_name"`
}

func (m *Model) ListLogFiles(ids []string) ([]*LogEntity, error) {
	logs := []*LogEntity{}

	if len(ids) == 0 {
		return logs, nil
	}

	idstr := `"` + strings.Join(ids, `","`) + `"`

	// TODO: avoid sql injection
	rows, err := m.db.Query(
		`SELECT id,instance_name FROM inspections WHERE id in (` + idstr + `) ORDER BY create_t DESC`,
	)
	if err != nil {
		log.Error("db.Query:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		l := LogEntity{}
		if err := rows.Scan(&l.Id, &l.InstanceName); err != nil {
			log.Error("db.Query:", err)
			return nil, err
		}
		logs = append(logs, &l)
	}

	return logs, nil
}

func (m *Model) ListLogInstances(ids []string) ([]*LogEntity, error) {
	logs := []*LogEntity{}

	if len(ids) == 0 {
		return logs, nil
	}

	idstr := `"` + strings.Join(ids, `","`) + `"`

	// TODO: avoid sql injection
	rows, err := m.db.Query(
		`SELECT id,name FROM instances WHERE id IN (` + idstr + `) ORDER BY create_t DESC`,
	)
	if err != nil {
		log.Error("db.Query:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		l := LogEntity{}
		if err := rows.Scan(&l.Id, &l.InstanceName); err != nil {
			log.Error("db.Query:", err)
			return nil, err
		}
		logs = append(logs, &l)
	}

	return logs, nil
}
