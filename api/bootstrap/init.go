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

	config := initConfig(path.Join(homepath, "tidb-foresight.toml"))

	db := initDB(path.Join(homepath, "sqlite.db"))

	return config, db
}
