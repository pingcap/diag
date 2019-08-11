package dmesg

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/dmesg"
	log "github.com/sirupsen/logrus"
)

type saveDmesgTask struct{}

func SaveDmesg() *saveDmesgTask {
	return &saveDmesgTask{}
}

// Save parsed dmesg logs to database
func (t *saveDmesgTask) Run(db *boot.DB, logs *dmesg.Dmesg, c *boot.Config) {
	for _, dmesg := range *logs {
		if _, err := db.Exec(
			`INSERT INTO inspection_dmesg(inspection, node_ip, log) VALUES(?, ?, ?)`,
			c.InspectionId, dmesg.Ip, dmesg.Log,
		); err != nil {
			log.Error("db.Exec:", err)
			return
		}
	}
}
