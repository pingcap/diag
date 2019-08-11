package items

import (
	log "github.com/sirupsen/logrus"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/input/status"
)

const (
	ITEM_BASIC   = "basic"
	ITEM_DBINFO  = "dbinfo"
	ITEM_METRIC  = "metric"
	ITEM_CONFIG  = "config"
	ITEM_PROFILE = "profile"
	ITEM_LOG     = "log"
)

type saveItemsTask struct{}

func SaveItems() *saveItemsTask {
	return &saveItemsTask{}
}

// Save the items and their result collector collected
func (t *saveItemsTask) Run(args *args.Args, c *boot.Config, db *boot.DB, s *status.StatusMap) {
	items := []string{
		ITEM_BASIC, ITEM_DBINFO, ITEM_METRIC, ITEM_CONFIG, ITEM_PROFILE, ITEM_LOG,
	}
	for _, item := range items {
		status := "none"
		message := ""
		if args.Collect(item) {
			if s.Get(item).Status == "success" {
				status = "running"
			} else {
				status = "exception"
				message = s.Get(item).Message
			}
		}

		if _, err := db.Exec(
			"REPLACE INTO inspection_items(inspection, name, status, message) VALUES(?, ?, ?, ?)",
			c.InspectionId, item, status, message,
		); err != nil {
			log.Error("db.Exec:", err)
			db.InsertSymptom(c.InspectionId, "exception", "write database", "contact developer")
		}
	}
}
