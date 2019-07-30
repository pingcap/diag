package model

import (
	"time"

	"github.com/pingcap/tidb-foresight/model/report"
	log "github.com/sirupsen/logrus"
)

type Inspection struct {
	Uuid       string      `json:"uuid"`
	InstanceId string      `json:"instance_id"`
	Status     string      `json:"status"`
	Message    string      `json:"message"`
	Type       string      `json:"type"`
	CreateTime *time.Time  `json:"create_time,omitempty"`
	FinishTime *time.Time  `json:"finish_time,omitempty"`
	ReportPath string      `json:"report_path"`
	Tidb       string      `json:"tidb"`
	Tikv       string      `json:"tikv"`
	Pd         string      `json:"pd"`
	Grafana    string      `json:"grafana"`
	Prometheus string      `json:"prometheus"`
	Report     interface{} `json:"report,omitempty"`
}

func (m *Model) ListAllInspections(page, size int64) ([]*Inspection, int, error) {
	inspections := []*Inspection{}

	rows, err := m.db.Query(
		"SELECT id,instance,status,message,type,create_t,tidb,tikv,pd,grafana,prometheus FROM inspections ORDER BY create_t DESC LIMIT ?, ?", 
		(page-1)*size, size,
	)
	if err != nil {
		log.Error("failed to call db.Query:", err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		inspection := Inspection{CreateTime: &time.Time{}, FinishTime: &time.Time{}}
		err := rows.Scan(
			&inspection.Uuid, &inspection.InstanceId, &inspection.Status, &inspection.Message,
			&inspection.Type, inspection.CreateTime, &inspection.Tidb, &inspection.Tikv, &inspection.Pd,
			&inspection.Grafana, &inspection.Prometheus,
		)
		if err != nil {
			log.Error("db.Query:", err)
			return nil, 0, err
		}

		report := report.NewReport(m.db, inspection.Uuid)
		err = report.Load()
		if err != nil {
			log.Error("load report:", err)
			return nil, 0, err
		}

		t := time.Now()
		inspection.FinishTime = &t // TODO: use real time
		inspection.ReportPath = "/api/v1/inspections/" + inspection.Uuid + ".tar.gz"
		inspection.Report = report
		inspections = append(inspections, &inspection)
	}

	total := 0
	if err = m.db.QueryRow("SELECT COUNT(id) FROM inspections").Scan(&total); err != nil {
		log.Error("db.Query:", err)
		return nil, 0, err
	}

	return inspections, total, nil
}

func (m *Model) ListInspections(instanceId string, page, size int64) ([]*Inspection, int, error) {
	inspections := []*Inspection{}

	rows, err := m.db.Query(
		"SELECT id,instance,status,message,type,create_t,tidb,tikv,pd,grafana,prometheus FROM inspections WHERE instance = ? limit ?,?",
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
			&inspection.Uuid, &inspection.InstanceId, &inspection.Status, &inspection.Message,
			&inspection.Type, &inspection.CreateTime, &inspection.Tidb, &inspection.Tikv, &inspection.Pd,
			&inspection.Grafana, &inspection.Prometheus,
		)
		if err != nil {
			log.Error("db.Query:", err)
			return nil, 0, err
		}

		report := report.NewReport(m.db, inspection.Uuid)
		err = report.Load()
		if err != nil {
			log.Error("load report:", err)
			return nil, 0, err
		}

		t := time.Now()
		inspection.FinishTime = &t // TODO: use real time
		inspection.ReportPath = "/api/v1/inspections/" + inspection.Uuid + ".tar.gz"
		inspection.Report = report
		inspections = append(inspections, &inspection)
	}

	total := 0
	if err = m.db.QueryRow("SELECT COUNT(id) FROM inspections WHERE instance = ?", instanceId).Scan(&total); err != nil {
		log.Error("db.Query:", err)
		return nil, 0, err
	}

	return inspections, total, nil
}

func (m *Model) SetInspection(inspection *Inspection) error {
	_, err := m.db.Exec(
		"REPLACE INTO inspections(id,instance,status,message,type,tidb,tikv,pd,grafana,prometheus) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		inspection.Uuid, inspection.InstanceId, inspection.Status, inspection.Message, inspection.Type,
		inspection.Tidb, inspection.Tikv, inspection.Pd, inspection.Grafana, inspection.Prometheus,
	)
	if err != nil {
		log.Error("db.Exec:", err)
		return err
	}

	return nil
}

func (m *Model) GetInspectionDetail(inspectionId string) (*Inspection, error) {
	inspection := Inspection{}
	err := m.db.QueryRow(
		"SELECT id,instance,status,message,type,create_t,tidb,tikv,pd,grafana,prometheus FROM inspections WHERE id = ?",
		inspectionId,
	).Scan(
		&inspection.Uuid, &inspection.InstanceId, &inspection.Status, &inspection.Message,
		&inspection.Type, &inspection.CreateTime, &inspection.Tidb, &inspection.Tikv, &inspection.Pd,
		&inspection.Grafana, &inspection.Prometheus,
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
