package slow_query

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type analyzeTask struct{}

func Analyze() *analyzeTask {
	return &analyzeTask{}
}

// Check if there is any slow query
func (t *analyzeTask) Run(m *boot.Model, c *boot.Config) {
	if querys, err := m.GetInspectionSlowLog(c.InspectionId); err != nil {
		log.Error("get slow log:", err)
		return
	} else if len(querys) > 0 {
		msg := "there are slow logs in the cluster"
		desc := "please check the slow log"
		m.InsertSymptom("warning", msg, desc)
	}
}
