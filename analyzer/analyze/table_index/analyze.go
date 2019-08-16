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
	tbs, err := m.GetTablesWithoutIndex(c.InspectionId)
	if err != nil {
		log.Error("get tables without index:", err)
		return
	}

	for _, tb := range tbs {
		m.InsertSymptom(
			"error",
			fmt.Sprintf("table %s missing index in database %s", tb.Table, tb.DB),
			"please add index for the table",
		)
	}
}
