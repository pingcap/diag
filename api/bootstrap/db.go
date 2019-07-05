package bootstrap

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

const INIT = `
CREATE TABLE IF NOT EXISTS config (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user VARCHAR(32),
	user_pass VARCHAR(32),
	admin VARCHAR(32),
	admin_pass VARCHAR(32),
	create_t DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS items (
	name VARCHAR(32),
	collect INT2,
	duration VARCHAR(32),
	PRIMARY KEY (name)
);

CREATE TABLE IF NOT EXISTS instances (
	id VARCHAR(64),
	name VARCHAR(32),
	status VARCHAR(32),
	create_t DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS instance_topology (
	instance_id VARCHAR(64),
	tidb VARCHAR(256),
	tikv VARCHAR(256),
	pd VARCHAR(256),
	grafana VARCHAR(256),
	prometheus VARCHAR(256)
);

CREATE TABLE IF NOT EXISTS inspections (
	id VARCHAR(64),
	instance VARCHAR(64),
	status VARCHAR(32),
	type VARCHAR(16),
	create_t DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS inspection_topology (
	inspection_id VARCHAR(64),
	tidb VARCHAR(256),
	tikv VARCHAR(256),
	pd VARCHAR(256),
	grafana VARCHAR(256),
	prometheus VARCHAR(256)
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
