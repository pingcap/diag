package model

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

type LogEntity struct {
	Id           string `json:"id"`
	InstanceName string `json:"instance_name"`
}

func (m *Model) ListLogFiles(ids []string, page, size int64) ([]*LogEntity, int, error) {
	logs := []*LogEntity{}

	if len(ids) == 0 {
		return logs, 0, nil
	}

	idstr := `"` + strings.Join(ids, `","`) + `"`

	// TODO: avoid sql injection
	rows, err := m.db.Query(
		`SELECT id,instance_name FROM inspections WHERE id in (`+idstr+`) ORDER BY create_t DESC limit ?, ?`,
		(page-1)*size, size,
	)
	if err != nil {
		log.Error("db.Query:", err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		l := LogEntity{}
		if err := rows.Scan(&l.Id, &l.InstanceName); err != nil {
			log.Error("db.Query:", err)
			return nil, 0, err
		}
		logs = append(logs, &l)
	}

	total := 0
	if err = m.db.QueryRow(`SELECT COUNT(*) FROM inspections WHERE id in (` + idstr + `)`).Scan(&total); err != nil {
		log.Error("db.Query:", err)
		return nil, 0, err
	}

	return logs, total, nil
}

func (m *Model) ListLogInstances(ids []string, page, size int64) ([]*LogEntity, int, error) {
	logs := []*LogEntity{}

	if len(ids) == 0 {
		return logs, 0, nil
	}

	idstr := `"` + strings.Join(ids, `","`) + `"`

	// TODO: avoid sql injection
	rows, err := m.db.Query(
		`SELECT id,name FROM instances WHERE id IN (`+idstr+`) ORDER BY create_t DESC limit ?, ?`,
		(page-1)*size, size,
	)
	if err != nil {
		log.Error("db.Query:", err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		l := LogEntity{}
		if err := rows.Scan(&l.Id, &l.InstanceName); err != nil {
			log.Error("db.Query:", err)
			return nil, 0, err
		}
		logs = append(logs, &l)
	}

	total := 0
	if err = m.db.QueryRow(`SELECT COUNT(*) FROM instances WHERE id in (` + idstr + `)`).Scan(&total); err != nil {
		log.Error("db.Query:", err)
		return nil, 0, err
	}

	return logs, total, nil
}
