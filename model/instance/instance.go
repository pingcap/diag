package instance

import (
	"time"
)

type Instance struct {
	Uuid       string    `json:"uuid" gorm:"PRIMARY_KEY"`
	Name       string    `json:"name"`
	User       string    `json:"user"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	CreateTime time.Time `json:"create_time"`
	Tidb       string    `json:"tidb"`
	Tikv       string    `json:"tikv"`
	Pd         string    `json:"pd"`
	Grafana    string    `json:"grafana"`
	Prometheus string    `json:"promethus"`
}

func (m *instance) ListInstance() ([]*Instance, error) {
	insts := []*Instance{}

	if err := m.db.Order("create_time desc").Find(&insts).Error(); err != nil {
		return nil, err
	}

	return insts, nil
}

func (m *instance) GetInstance(instanceId string) (*Instance, error) {
	inst := Instance{}

	if err := m.db.Where(&Instance{Uuid: instanceId}).Take(&inst).Error(); err != nil {
		return nil, err
	}

	return &inst, nil
}

func (m *instance) CreateInstance(inst *Instance) error {
	return m.db.Create(inst).Error()
}

func (m *instance) UpdateInstance(inst *Instance) error {
	return m.db.Model(&Instance{}).Updates(inst).Error()
}

func (m *instance) DeleteInstance(uuid string) error {
	return m.db.Delete(&Instance{Uuid: uuid}).Error()
}
