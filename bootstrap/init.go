package bootstrap

import (
	"os"
	"path"

	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

func MustInit(homepath string) (*ForesightConfig, db.DB) {
	if err := os.MkdirAll(homepath, os.ModePerm); err != nil {
		log.Panic("can't access home: ", homepath)
	}

	webDir := path.Join(homepath, "web")
	if err := os.MkdirAll(webDir, os.ModePerm); err != nil {
		log.Panic("can't access web dir: ", webDir)
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
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Panic("can't access log dir: ", logDir)
	}

	profileDir := path.Join(homepath, "profile")
	if err := os.MkdirAll(profileDir, os.ModePerm); err != nil {
		log.Panic("can't access profile dir: ", profileDir)
	}

	config := initConfig(homepath)

	db := initDB(path.Join(homepath, "sqlite.db"))

	return config, db
}
