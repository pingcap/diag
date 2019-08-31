package status

import (
	"fmt"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/status"
)

type analyzeTask struct{}

func Analyze() *analyzeTask {
	return &analyzeTask{}
}

// Check if there is any slow query
func (t *analyzeTask) Run(m *boot.Model, sm *status.StatusMap) {
	for item, s := range *sm {
		if s.Status != "success" {
			msg := fmt.Sprintf("collect %s failed", item)
			desc := s.Message
			m.InsertSymptom("exception", msg, desc)
		}
	}
}
