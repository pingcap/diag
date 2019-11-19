package hardware

import (
	"encoding/json"
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

		// TODO: this fucking messages are only for fucking debugging.
		fuckingDistStorage, err := json.Marshal(insight.Sysinfo)
		if err == nil {
			log.Infof("Collected message for %s", string(fuckingDistStorage))
		}

		for _, disk := range insight.Sysinfo.Storage {
			marked := false
			for _, hardDisks := range insight.BlockInfo.Disks {
				if hardDisks.Name == disk.Name {
					disks = append(disks, fmt.Sprintf("Disk type: %s, disk_controller: %s, %s(%s)",
						hardDisks.DriveType, hardDisks.StorageController, disk.Name, disk.Driver))
					marked = true
				}
			}
			if !marked {
				disks = append(disks, fmt.Sprintf("%s(%s)", disk.Name, disk.Driver))
			}
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
