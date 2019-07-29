package model

import (
	"time"

	log "github.com/sirupsen/logrus"
)

type Instance struct {
	Uuid       string    `json:"uuid"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	CreateTime time.Time `json:"create_time"`
	Tidb       string    `json:"tidb"`
	Tikv       string    `json:"tikv"`
	Pd         string    `json:"pd"`
	Grafana    string    `json:"grafana"`
	Prometheus string    `json:"promethus"`
}

func (m *Model) ListInstance() ([]*Instance, error) {
	instances := []*Instance{}

	rows, err := m.db.Query("SELECT id,name,status,message,create_t,tidb,tikv,pd,grafana,prometheus FROM instances")
	if err != nil {
		log.Error("Failed to call db.Query:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		instance := Instance{}
		err := rows.Scan(
			&instance.Uuid, &instance.Name, &instance.Status, &instance.Message, &instance.CreateTime,
			&instance.Tidb, &instance.Tikv, &instance.Pd, &instance.Grafana, &instance.Prometheus,
		)
		if err != nil {
			log.Error("db.Query:", err)
			return nil, err
		}

		instances = append(instances, &instance)
	}

	return instances, nil
}

func (m *Model) GetInstance(instanceId string) (*Instance, error) {
	instance := Instance{}
	err := m.db.QueryRow(
		"SELECT id,name,status,message,create_t,tidb,tikv,pd,grafana,prometheus FROM instances WHERE id = ?",
		instanceId,
	).Scan(
		&instance.Uuid, &instance.Name, &instance.Status, &instance.Message, &instance.CreateTime,
		&instance.Tidb, &instance.Tikv, &instance.Pd, &instance.Grafana, &instance.Prometheus,
	)
	if err != nil {
		log.Error("db.Query:", err)
		return nil, err
	}
	return &instance, nil
}

func (m *Model) CreateInstance(instance *Instance) error {
	_, err := m.db.Exec(
		"INSERT INTO instances(id,name,status,tidb,tikv,pd,grafana,prometheus) VALUES(?, ?, ?, ?, ?, ?, ?, ?)",
		instance.Uuid, instance.Name, "pending", instance.Tidb,
		instance.Tikv, instance.Pd, instance.Grafana, instance.Prometheus,
	)
	if err != nil {
		log.Error("failed to call db.Exec:", err)
		return err
	}

	return nil
}

func (m *Model) UpdateInstance(instance *Instance) error {
	_, err := m.db.Exec(
		"UPDATE instances SET name=?,status=?,message=?,tidb=?,tikv=?,pd=?,grafana=?,prometheus=? WHERE id=?",
		instance.Name, instance.Status, instance.Message, instance.Tidb, instance.Tikv,
		instance.Pd, instance.Grafana, instance.Prometheus, instance.Uuid,
	)
	if err != nil {
		log.Error("failed to call db.Exec:", err)
		return err
	}

	return nil
}

func (m *Model) DeleteInstance(uuid string) error {
	_, err := m.db.Exec("DELETE FROM instances WHERE id = ?", uuid)
	if err != nil {
		log.Error("Failed to call db.Exec:", err)
		return err
	}

	return nil
}
