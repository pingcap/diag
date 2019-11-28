package worker

import (
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
)

type Worker interface {
	Collect(inspectionId string, config *model.Config, env map[string]string) error
	Analyze(inspectionId string) error
}

type worker struct {
	c *bootstrap.ForesightConfig
	m model.Model
}

func New(c *bootstrap.ForesightConfig, m model.Model) Worker {
	return &worker{c, m}
}
