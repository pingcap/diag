package alert

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type analyzeTask struct{}

func Analyze() *analyzeTask {
	return &analyzeTask{}
}

// Check if any table doest not have index
func (t *analyzeTask) Run(m *boot.Model, c *boot.Config) {
	tbs, err := m.GetInspectionDBInfo(c.InspectionId)
	if err != nil {
		log.Error("get tables index:", err)
		return
	}

	cm := make(map[string]int, 0)
	for _, tb := range tbs {
		if tb.Index.GetValue() == 0 {
			cm[tb.DB]++
		}
	}
	for db, count := range cm {
		m.InsertSymptom(
			"error",
			fmt.Sprintf("there are %d tables missing index in database %s", count, db),
			"please add index for these tables",
		)
	}
}
