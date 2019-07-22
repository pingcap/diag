package bootstrap

import (
	"database/sql"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

func MustInit(homepath string) (*ForesightConfig, *sql.DB) {
	if err := os.MkdirAll(homepath, os.ModePerm); err != nil {
		log.Panic("can't access home: ", homepath)
	}

	staticDir := path.Join(homepath, "static")
	if err := os.MkdirAll(staticDir, os.ModePerm); err != nil {
		log.Panic("can't access static dir: ", staticDir)
	}

	inventoryDir := path.Join(homepath, "inventory")
	if err := os.MkdirAll(inventoryDir, os.ModePerm); err != nil {
		log.Panic("can't access inventory dir: ", inventoryDir)
	}

	topologyDir := path.Join(homepath, "topology")
	if err := os.MkdirAll(topologyDir, os.ModePerm); err != nil {
		log.Panic("can't access topology dir: ", topologyDir)
	}

	inspectionDir := path.Join(homepath, "inspection")
	if err := os.MkdirAll(inspectionDir, os.ModePerm); err != nil {
		log.Panic("can't access inspection dir: ", inspectionDir)
	}

	packageDir := path.Join(homepath, "package")
	if err := os.MkdirAll(packageDir, os.ModePerm); err != nil {
		log.Panic("can't access package dir: ", packageDir)
	}

	logDir := path.Join(homepath, "remote-log")
	if err := os.MkdirAll(packageDir, os.ModePerm); err != nil {
		log.Panic("can't access log dir: ", logDir)
	}

	config := initConfig(path.Join(homepath, "tidb-foresight.toml"))

	db := initDB(path.Join(homepath, "sqlite.db"))

	return config, db
}
