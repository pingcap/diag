package bootstrap

import (
	_ "github.com/mattn/go-sqlite3"
	"github.com/pingcap/tidb-foresight/wrapper/db"
	log "github.com/sirupsen/logrus"
)

func initDB(dbpath string) db.DB {
	db, err := db.Open(db.SQLITE, dbpath)
	//db, err := db.OpenDebug(db.SQLITE, dbpath)

	if err != nil {
		log.Panic(err)
	}

	return db
}
