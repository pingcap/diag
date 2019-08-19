package logs

import (
	"github.com/pingcap/tidb-foresight/wraper/db"
)

type Model interface {
	ListLogFiles(ids []string) ([]*LogEntity, error)
	ListLogInstances(ids []string) ([]*LogEntity, error)
}

func New(db db.DB) Model {
	return &logs{db}
}

type logs struct {
	db db.DB
}
