package utils

import (
	"github.com/pingcap/tidb-foresight/wrapper/db"
	log "github.com/sirupsen/logrus"
)

func MustInitSchema(db db.DB, schemas ...interface{}) {
	for _, schema := range schemas {
		if !db.HasTable(schema) {
			if err := db.CreateTable(schema).Error(); err != nil {
				log.Panic("init schema:", err)
			}
		}
		db.Debug().AutoMigrate(schemas)
	}
}
