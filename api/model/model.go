package model

import (
	"database/sql"
)

type Model struct {
	db *sql.DB
}

func NewModel(db *sql.DB) *Model {
	return &Model{
		db: db,
	}
}
