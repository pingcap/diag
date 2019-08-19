package instance

import (
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wraper/db"
)

type Model interface {
	ListInstance() ([]*Instance, error)
	GetInstance(instanceId string) (*Instance, error)
	CreateInstance(inst *Instance) error
	UpdateInstance(inst *Instance) error
	DeleteInstance(uuid string) error
}

func New(db db.DB) Model {
	utils.MustInitSchema(db, &Instance{})
	return &instance{db}
}

type instance struct {
	db db.DB
}
