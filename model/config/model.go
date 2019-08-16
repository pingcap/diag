package config

import (
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wraper/db"
)

type Model interface {
	GetInstanceConfig(instanceId string) (*Config, error)
	SetInstanceConfig(c *Config) error
	DeleteInstanceConfig(instanceId string) error
}

func New(db db.DB) Model {
	utils.MustInitSchema(db, &Config{})
	return &config{db}
}

type config struct {
	db db.DB
}
