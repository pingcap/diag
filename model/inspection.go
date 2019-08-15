package model

import (
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type Inspection struct {
	Uuid         string         `json:"uuid"`
	InstanceId   string         `json:"instance_id"`
	InstanceName string         `json:"instance_name"`
	User         string         `json:"user"`
	Status       string         `json:"status"`
	Message      string         `json:"message"`
	Type         string         `json:"type"`
	CreateTime   utils.NullTime `json:"create_time,omitempty"`
	FinishTime   utils.NullTime `json:"finish_time,omitempty"`
	ScrapeBegin  utils.NullTime `json:"scrape_begin,omitempty"`
	ScrapeEnd    utils.NullTime `json:"scrape_end,omitempty"`
	Tidb         string         `json:"tidb"`
	Tikv         string         `json:"tikv"`
	Pd           string         `json:"pd"`
	Grafana      string         `json:"grafana"`
	Prometheus   string         `json:"prometheus"`
}

func (m *Model) ListAllInspections(page, size int64) ([]*Inspection, int, error) {
	inspections := []*Inspection{}

	rows, err := m.db.Query(
		`SELECT id,instance,instance_name,user,status,message,type,create_t,finish_t,tidb,tikv,pd,grafana,prometheus 
		FROM inspections WHERE type IN ('manual', 'auto') ORDER BY create_t DESC LIMIT ?, ?`,
		(page-1)*size, size,
	)
	if err != nil {
		log.Error("failed to call db.Query:", err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		inspection := Inspection{}
		err := rows.Scan(
			&inspection.Uuid, &inspection.InstanceId, &inspection.InstanceName, &inspection.User, &inspection.Status,
			&inspection.Message, &inspection.Type, &inspection.CreateTime, &inspection.FinishTime, &inspection.Tidb,
			&inspection.Tikv, &inspection.Pd, &inspection.Grafana, &inspection.Prometheus,
		)
		if err != nil {
			log.Error("db.Query:", err)
			return nil, 0, err
		}

		inspections = append(inspections, &inspection)
	}

	total := 0
	if err = m.db.QueryRow("SELECT COUNT(id) FROM inspections WHERE type IN ('manual', 'auto')").Scan(&total); err != nil {
		log.Error("db.Query:", err)
		return nil, 0, err
	}

	return inspections, total, nil
}

func (m *Model) ListInspections(instanceId string, page, size int64) ([]*Inspection, int, error) {
	inspections := []*Inspection{}

	rows, err := m.db.Query(
		`SELECT id,instance,instance_name,user,status,message,type,create_t,finish_t,scrape_bt,scrape_et,tidb,tikv,pd,grafana,prometheus 
		FROM inspections WHERE instance = ? AND type IN ('manual', 'auto') ORDER BY create_t DESC LIMIT ?, ?`,
		instanceId, (page-1)*size, size,
	)
	if err != nil {
		log.Error("Failed to call db.Query:", err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		inspection := Inspection{}
		err := rows.Scan(
			&inspection.Uuid, &inspection.InstanceId, &inspection.InstanceName, &inspection.User, &inspection.Status,
			&inspection.Message, &inspection.Type, &inspection.CreateTime, &inspection.FinishTime, &inspection.ScrapeBegin,
			&inspection.ScrapeEnd, &inspection.Tidb, &inspection.Tikv, &inspection.Pd, &inspection.Grafana, &inspection.Prometheus,
		)
		if err != nil {
			log.Error("db.Query:", err)
			return nil, 0, err
		}

		inspections = append(inspections, &inspection)
	}

	total := 0
	if err = m.db.QueryRow(
		"SELECT COUNT(id) FROM inspections WHERE instance = ? AND type IN ('manual', 'auto')",
		instanceId,
	).Scan(&total); err != nil {
		log.Error("db.Query:", err)
		return nil, 0, err
	}

	return inspections, total, nil
}

func (m *Model) SetInspection(inspection *Inspection) error {
	_, err := m.db.Exec(
		`REPLACE INTO inspections(id,instance,instance_name,user,status,message,type,tidb,tikv,pd,grafana,prometheus) 
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inspection.Uuid, inspection.InstanceId, inspection.InstanceName, inspection.User,
		inspection.Status, inspection.Message, inspection.Type, inspection.Tidb, inspection.Tikv,
		inspection.Pd, inspection.Grafana, inspection.Prometheus,
	)
	if err != nil {
		log.Error("db.Exec:", err)
		return err
	}

	return nil
}

func (m *Model) GetInspection(inspectionId string) (*Inspection, error) {
	inspection := Inspection{}
	err := m.db.QueryRow(
		`SELECT id,instance,instance_name,user,status,message,type,create_t,finish_t,scrape_bt,scrape_et,
		tidb,tikv,pd,grafana,prometheus FROM inspections WHERE id = ?`,
		inspectionId,
	).Scan(
		&inspection.Uuid, &inspection.InstanceId, &inspection.InstanceName, &inspection.User, &inspection.Status,
		&inspection.Message, &inspection.Type, &inspection.CreateTime, &inspection.FinishTime, &inspection.ScrapeBegin,
		&inspection.ScrapeEnd, &inspection.Tidb, &inspection.Tikv, &inspection.Pd, &inspection.Grafana, &inspection.Prometheus,
	)
	if err != nil {
		log.Error("db.Query:", err)
		return nil, err
	}

	return &inspection, nil
}

func (m *Model) DeleteInspection(inspectionId string) error {
	_, err := m.db.Exec("DELETE FROM inspections WHERE id = ?", inspectionId)
	if err != nil {
		log.Error("db.Exec:", err)
		return err
	}

	return nil
}
