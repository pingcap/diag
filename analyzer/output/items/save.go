package items

import (
	log "github.com/sirupsen/logrus"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/args"
	"github.com/pingcap/tidb-foresight/analyzer/input/status"
	"github.com/pingcap/tidb-foresight/model"
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
func (t *saveItemsTask) Run(args *args.Args, c *boot.Config, m *boot.Model, s *status.StatusMap) {
	items := []string{
		ITEM_BASIC, ITEM_DBINFO, ITEM_METRIC, ITEM_CONFIG, ITEM_PROFILE, ITEM_LOG,
	}
	for _, item := range items {
		status := "none"
		message := ""
		if args.Collect(item) {
			if s.Get(item).Status == "success" {
				status = "success"
			} else {
				status = "exception"
				message = s.Get(item).Message
			}
		}

		if err := m.InsertInspectionItem(&model.Item{
			InspectionId: c.InspectionId,
			Name:         item,
			Status:       status,
			Messages:     message,
		}); err != nil {
			log.Error("insert inspection item:", err)
			return
		}
	}
}
