package bootstrap

import (
	_ "github.com/mattn/go-sqlite3"
	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

func initDB(dbpath string) db.DB {
	db, err := db.Open(dbpath)

	if err != nil {
		log.Panic(err)
	}

	return db
}
