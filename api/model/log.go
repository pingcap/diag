package model

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

type LogEntity struct {
	Id           string `json:"id"`
	InstanceName string `json:"instance_name"`
}

func (m *Model) loadLogsFromDB(query string) ([]*LogEntity, error) {
	logs := []*LogEntity{}

	// TODO: avoid sql injection
	rows, err := m.db.Query(query)
	if err != nil {
		log.Error("Failed to call db.Query:", err)
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

func (m *Model) ListLogs(ids []string) ([]*LogEntity, error) {
	logs := []*LogEntity{}

	if len(ids) == 0 {
		return logs, nil
	}

	// TODO: avoid sql injection
	qinstance := `SELECT id,name FROM instances WHERE id IN("` + strings.Join(ids, `","`) + `")`
	qinspection := `SELECT id,instance_name FROM inspections WHERE id IN("` + strings.Join(ids, `","`) + `")`

	if ls, err := m.loadLogsFromDB(qinstance); err != nil {
		logs = append(logs, ls...)
	} else {
		return nil, err
	}
	if ls, err := m.loadLogsFromDB(qinspection); err != nil {
		logs = append(logs, ls...)
	} else {
		return nil, err
	}
	return logs, nil
}
