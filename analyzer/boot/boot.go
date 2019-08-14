package boot

import (
	"path"

	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

const (
	SQLITE = "sqlite3"
)

type bootstrapTask struct {
	inspectionId string
	home         string
	db           db.DB
}

// Generate config and connect database
func Bootstrap(inspectionId, home string) *bootstrapTask {
	db, err := db.Open(path.Join(home, "sqlite.db"))
	if err != nil {
		log.Panic("connection database:", err)
	}

	return &bootstrapTask{inspectionId, home, db}
}

func (t *bootstrapTask) Run() (*Config, *DB) {
	return newConfig(t.inspectionId, t.home), newDB(t.db)
}
