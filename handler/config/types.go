package config

import (
	"github.com/pingcap/tidb-foresight/model"
)

type ConfigSeter interface {
	SetInstanceConfig(config *model.Config) error
}

type ConfigGeter interface {
	GetInstanceConfig(instanceId string) (*model.Config, error)
}
