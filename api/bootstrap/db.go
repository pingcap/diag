package bootstrap

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

const INIT = `
CREATE TABLE IF NOT EXISTS items (
	name VARCHAR(32),
	collect INT2,
	duration VARCHAR(32),
	PRIMARY KEY (name)
);

CREATE TABLE IF NOT EXISTS instances (
	id VARCHAR(64) NOT NULL,
	name VARCHAR(32) NOT NULL,
	status VARCHAR(32) NOT NULL,
	user VARCHAR(32) NOT NULL DEFAULT "",
	tidb VARCHAR(256) NOT NULL DEFAULT "",
	tikv VARCHAR(256) NOT NULL DEFAULT "",
	pd VARCHAR(256) NOT NULL DEFAULT "",
	grafana VARCHAR(256) NOT NULL DEFAULT "",
	prometheus VARCHAR(256) NOT NULL DEFAULT "",
	create_t DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS inspections (
	id VARCHAR(64),
	instance VARCHAR(64),
	status VARCHAR(32),
	type VARCHAR(16),
	tidb VARCHAR(256),
	tikv VARCHAR(256),
	pd VARCHAR(256),
	grafana VARCHAR(256),
	prometheus VARCHAR(256),
	create_t DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

func initDB(dbpath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		log.Panic("open database failed:", dbpath)
	}

	if _, err = db.Exec(INIT); err != nil {
		log.Panic("execute initial statement failed")
	}

	return db
}
