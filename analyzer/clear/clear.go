package clear

import (
	log "github.com/sirupsen/logrus"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
)

type clearHistoryTask struct{}

func ClearHistory() *clearHistoryTask {
	return &clearHistoryTask{}
}

// Delete records having the same inspection id for idempotency
func (t *clearHistoryTask) Run(c *boot.Config, m *boot.Model) {
	if err := m.ClearInspectionItem(c.InspectionId); err != nil {
		log.Error("clear inspection item:", err)
		return
	}

	if err := m.ClearInspectionSymptom(c.InspectionId); err != nil {
		log.Error("clear inspection symptom:", err)
		return
	}

	if err := m.ClearInspectionBasicInfo(c.InspectionId); err != nil {
		log.Error("clear inspection basic info:", err)
		return
	}

	if err := m.ClearInspectionDBInfo(c.InspectionId); err != nil {
		log.Error("clear inspection dbinfo:", err)
		return
	}

	if err := m.ClearInspectionSlowLog(c.InspectionId); err != nil {
		log.Error("clear inspection slow log:", err)
		return
	}

	if err := m.ClearInspectionNetworkInfo(c.InspectionId); err != nil {
		log.Error("clear inspection network info:", err)
		return
	}

	if err := m.ClearInspectionAlertInfo(c.InspectionId); err != nil {
		log.Error("clear inspection alert info:", err)
		return
	}

	if err := m.ClearInspectionHardwareInfo(c.InspectionId); err != nil {
		log.Error("clear inspection hardware info:", err)
		return
	}

	if err := m.ClearInspectionDmesgLog(c.InspectionId); err != nil {
		log.Error("clear inspection dmesg log:", err)
		return
	}

	if err := m.ClearInspectionSoftwareInfo(c.InspectionId); err != nil {
		log.Error("clear inspection software info:", err)
		return
	}

	if err := m.ClearInspectionConfigInfo(c.InspectionId); err != nil {
		log.Error("clear inspection config info:", err)
		return
	}

	if err := m.ClearInspectionResourceInfo(c.InspectionId); err != nil {
		log.Error("clear inspection resource info:", err)
		return
	}

	if err := m.ClearInspectionNtpInfo(c.InspectionId); err != nil {
		log.Error("clear inspection ntp info:", err)
		return
	}

	if err := m.ClearInspectionTopologyInfo(c.InspectionId); err != nil {
		log.Error("clear inspection topology info:", err)
		return
	}
}
