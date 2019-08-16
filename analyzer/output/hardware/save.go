package hardware

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/insight"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type saveHardwareInfoTask struct{}

func SaveHardwareInfo() *saveHardwareInfoTask {
	return &saveHardwareInfoTask{}
}

// Save hardware info to database, the hardware info comes from insight collector
func (t *saveHardwareInfoTask) Run(m *boot.Model, c *boot.Config, insight *insight.Insight) {
	for _, insight := range *insight {
		disks := []string{}
		for _, disk := range insight.Sysinfo.Storage {
			disks = append(disks, disk.Name)
		}
		networks := []string{}
		for _, network := range insight.Sysinfo.Network {
			networks = append(networks, fmt.Sprintf("%s:%d", network.Name, network.Speed))
		}

		if err := m.InsertInspectionHardwareInfo(&model.HardwareInfo{
			InspectionId: c.InspectionId,
			NodeIp:       insight.NodeIp,
			Cpu:          insight.Sysinfo.Cpu.Model,
			Memory:       insight.Sysinfo.Memory.Type,
			Disk:         strings.Join(disks, ","),
			Network:      strings.Join(networks, ","),
		}); err != nil {
			log.Error("insert hardware info:", err)
			return
		}
	}
}
