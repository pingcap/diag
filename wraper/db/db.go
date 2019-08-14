package db

import (
	"database/sql"
)

const (
	SQLITE = "sqlite3"
)

// For decoupling with sql.DB to be friendly with unit test
type DB interface {
	Exec(query string, args ...interface{}) (Result, error)
	Query(query string, args ...interface{}) (Rows, error)
	QueryRow(query string, args ...interface{}) Row
	Close() error
}

// Open sqlite.db and return DB interface instead of a struct
func Open(fp string) (DB, error) {
	if ins, err := sql.Open(SQLITE, fp); err == nil {
		return &wrapedDB{ins}, nil
	} else {
		return nil, err
	}
}

type wrapedDB struct {
	instance *sql.DB
}

func (db *wrapedDB) Exec(query string, args ...interface{}) (Result, error) {
	return db.instance.Exec(query, args...)
}

func (db *wrapedDB) Query(query string, args ...interface{}) (Rows, error) {
	return db.instance.Query(query, args...)
}

func (db *wrapedDB) QueryRow(query string, args ...interface{}) Row {
	return db.instance.QueryRow(query, args...)
}

func (db *wrapedDB) Close() error {
	return db.Close()
}
