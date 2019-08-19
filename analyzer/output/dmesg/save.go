package dmesg

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/dmesg"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type saveDmesgTask struct{}

func SaveDmesg() *saveDmesgTask {
	return &saveDmesgTask{}
}

// Save parsed dmesg logs to database
func (t *saveDmesgTask) Run(m *boot.Model, logs *dmesg.Dmesg, c *boot.Config) {
	for _, dmesg := range *logs {
		if err := m.InsertInspectionDmesgLog(&model.DmesgLog{
			InspectionId: c.InspectionId,
			NodeIp:       dmesg.Ip,
			Log:          dmesg.Log,
		}); err != nil {
			log.Error("insert dmesg:", err)
			return
		}
	}
}
