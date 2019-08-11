package hardware

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/insight"
	log "github.com/sirupsen/logrus"
)

type saveHardwareInfoTask struct{}

func SaveHardwareInfo() *saveHardwareInfoTask {
	return &saveHardwareInfoTask{}
}

// Save hardware info to database, the hardware info comes from insight collector
func (t *saveHardwareInfoTask) Run(db *boot.DB, c *boot.Config, insight *insight.Insight) {
	for _, insight := range *insight {
		nodeIp := insight.NodeIp
		cpu := insight.Sysinfo.Cpu.Model
		memory := insight.Sysinfo.Memory.Type
		disks := []string{}
		for _, disk := range insight.Sysinfo.Storage {
			disks = append(disks, disk.Name)
		}
		networks := []string{}
		for _, network := range insight.Sysinfo.Network {
			networks = append(networks, fmt.Sprintf("%s:%d", network.Name, network.Speed))
		}
		if _, err := db.Exec(
			`INSERT INTO inspection_hardware(inspection, node_ip, cpu, memory, disk, network) VALUES(?, ?, ?, ?, ?, ?)`,
			c.InspectionId, nodeIp, cpu, memory, strings.Join(disks, ","), strings.Join(networks, ","),
		); err != nil {
			log.Error("db.Exec:", err)
			return
		}
	}
}
