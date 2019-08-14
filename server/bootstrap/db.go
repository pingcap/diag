package bootstrap

import (
	_ "github.com/mattn/go-sqlite3"
	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

const INIT = `
CREATE TABLE IF NOT EXISTS instances (
	id VARCHAR(64) NOT NULL,
	name VARCHAR(32) NOT NULL,
	user VARCHAR(32) NOT NULL,
	status VARCHAR(32) NOT NULL,
	message VARCHAR(256) NOT NULL DEFAULT "",
	tidb VARCHAR(256) NOT NULL DEFAULT "",
	tikv VARCHAR(256) NOT NULL DEFAULT "",
	pd VARCHAR(256) NOT NULL DEFAULT "",
	grafana VARCHAR(256) NOT NULL DEFAULT "",
	prometheus VARCHAR(256) NOT NULL DEFAULT "",
	create_t DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS configs (
	instance VARCHAR(64),
	c_hardw INT2,
	c_softw INT2,
	c_log INT2,
	c_demsg INT2,
	s_start VARCHAR(32),
	s_duration INT,
	r_duration VARCHAR(16),
	PRIMARY KEY (instance)
);

CREATE TABLE IF NOT EXISTS inspections (
	id VARCHAR(64) NOT NULL,
	instance VARCHAR(64) NOT NULL,
	instance_name VARCHAR(64) NOT NULL DEFAULT "",
	user VARCHAR(32) NOT NULL,
	status VARCHAR(32) NOT NULL,
	message TEXT NOT NULL DEFAULT "",
	type VARCHAR(16) NOT NULL,
	tidb VARCHAR(256) NOT NULL DEFAULT "",
	tikv VARCHAR(256) NOT NULL DEFAULT "",
	pd VARCHAR(256) NOT NULL DEFAULT "",
	grafana VARCHAR(256) NOT NULL DEFAULT "",
	prometheus VARCHAR(256) NOT NULL DEFAULT "",
	create_t DATETIME DEFAULT CURRENT_TIMESTAMP,
	finish_t DATETIME DEFAULT NULL,
	scrape_bt DATETIME DEFAULT NULL,
	scrape_et DATETIME DEFAULT NULL,
	PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS inspection_items (
	inspection VARCHAR(64) NOT NULL,
	name VARCHAR(32) NOT NULL,
	status VARCHAR(32) NOT NULL,
	message VARCHAR(1024) NOT NULL DEFAULT "",
	PRIMARY KEY (inspection, name)
);

CREATE TABLE IF NOT EXISTS inspection_symptoms (
	inspection VARCHAR(64) NOT NULL,
	status VARCHAR(16) NOT NULL,
	message TEXT NOT NULL DEFAULT "",
	description TEXT NOT NULL DEFAULT ""
);

CREATE TABLE IF NOT EXISTS inspection_basic_info (
	inspection VARCHAR(64),
	cluster_name VARCHAR(64),
	cluster_create_t  DATETIME DEFAULT CURRENT_TIMESTAMP,
	inspect_t DATETIME DEFAULT CURRENT_TIMESTAMP,
	tidb_count INT,
	tikv_count INT,
	pd_count INT,
	PRIMARY KEY (inspection)
);

CREATE TABLE IF NOT EXISTS inspection_db_info (
	inspection VARCHAR(64) NOT NULL,
	db VARCHAR(64) NOT NULL,
	tb VARCHAR(64) NOT NULL,
	idx int NOT NULL DEFAULT 0,
	PRIMARY KEY (inspection, db, tb)
);

CREATE TABLE IF NOT EXISTS inspection_slow_log (
	inspection VARCHAR(64) NOT NULL,
	instance VARCHAR(64) NOT NULL,
	time DATETIME NOT NULL,
	txn_start_ts INT64 NOT NULL,
	user VARCHAR(64) NOT NULL,
	conn_id INT64 UNSIGNED NOT NULL,
	query_time DOUBLE NOT NULL,
	db VARCHAR(64) NOT NULL,
	digest VARCHAR(64) NOT NULL,
	query TEXT NOT NULL,
	node_ip VARCHAR(16) NOT NULL
);

CREATE TABLE IF NOT EXISTS inspection_network (
	inspection VARCHAR(64) NOT NULL,
	node_ip VARCHAR(16) NOT NULL,
	connections INT64 NOT NULL,
	recv INT64 NOT NULL,
	send INT64 NOT NULL,
	bad_seg INT64 NOT NULL,
	retrans INT64 NOT NULL
);

CREATE TABLE IF NOT EXISTS inspection_alerts (
	inspection VARCHAR(64) NOT NULL,
	name VARCHAR(64) NOT NULL,
	value TEXT NOT NULL,
	time DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS inspection_hardware (
	inspection VARCHAR(64) NOT NULL,
	node_ip VARCHAR(16) NOT NULL,
	cpu VARCHAR(64) NOT NULL,
	memory VARCHAR(64) NOT NULL,
	disk VARCHAR(64) NOT NULL,
	network VARCHAR(64) NOT NULL,
	PRIMARY KEY (inspection, node_ip)
);

CREATE TABLE IF NOT EXISTS inspection_dmesg (
	inspection VARCHAR(64) NOT NULL,
	node_ip VARCHAR(16) NOT NULL,
	log TEXT NOT NULL,
	PRIMARY KEY (inspection, node_ip)
);

CREATE TABLE IF NOT EXISTS software_version (
	inspection VARCHAR(64) NOT NULL,
	node_ip VARCHAR(16) NOT NULL,
	component VARCHAR(16) NOT NULL,
	version VARCHAR(64) NOT NULL
);

CREATE TABLE IF NOT EXISTS software_config (
	inspection VARCHAR(64) NOT NULL,
	node_ip VARCHAR(16) NOT NULL,
	port INT NOT NULL,
	component VARCHAR(16) NOT NULL,
	config TEXT NOT NULL,
	PRIMARY KEY (inspection, node_ip, port, component)
);

CREATE TABLE IF NOT EXISTS inspection_resource (
	inspection VARCHAR(64) NOT NULL,
	resource VARCHAR(64) NOT NULL,
	duration VARCHAR(64) NOT NULL,
	value DOUBLE NOT NULL
);
`

func initDB(dbpath string) db.DB {
	db, err := db.Open(dbpath)
	if err != nil {
		log.Panicf("open database(%s) failed: %s", dbpath, err)
	}

	if _, err = db.Exec(INIT); err != nil {
		log.Panic("execute initial statement failed: ", err)
	}

	return db
}
