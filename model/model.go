package model

import (
	"github.com/pingcap/tidb-foresight/wraper/db"
)

type Model struct {
	db db.DB
}

func NewModel(db db.DB) *Model {
	return &Model{
		db: db,
	}
}
