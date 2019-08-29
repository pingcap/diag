package worker

import (
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
)

type Worker interface {
	Collect(inspectionId, inspectionType string, config *model.Config) error
	Analyze(inspectionId string) error
}

type worker struct {
	c *bootstrap.ForesightConfig
	m model.Model
}

func New(c *bootstrap.ForesightConfig, m model.Model) Worker {
	return &worker{c, m}
}
