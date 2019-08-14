package boot

import (
	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

// with usefull tool functions
type DB struct {
	db.DB
}

func newDB(db db.DB) *DB {
	return &DB{
		db,
	}
}

// Insert a symptom for report.
// status: info, warning, error, exception
// message: descript what happend
// description: tell user what he can do
func (db *DB) InsertSymptom(inspectionId, status, message, description string) {
	if _, err := db.Exec(
		"INSERT INTO inspection_symptoms(inspection, status, message, description) VALUES(?, ?, ?, ?)",
		inspectionId, status, message, description,
	); err != nil {
		log.Panic("insert symptom:", err)
	}
}
