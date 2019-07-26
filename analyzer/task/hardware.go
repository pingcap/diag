package task

import (
	"fmt"
	"strings"
	"database/sql"

	log "github.com/sirupsen/logrus"
)

type SaveHardwareInfoTask struct {
	BaseTask
}

func SaveHardwareInfo(inspectionId string, src string, data *TaskData, db *sql.DB) Task {
	return &SaveHardwareInfoTask {BaseTask{inspectionId, src, data, db}}
}

func (t *SaveHardwareInfoTask) Run() error {
	if !t.data.collect[ITEM_BASIC] || t.data.status[ITEM_BASIC].Status != "success" {
		return nil
	}

	for _, insight := range t.data.insight {
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
		if _, err := t.db.Exec(
			`INSERT INTO inspection_hardware(inspection, cpu, memory, disk, network) VALUES(?, ?, ?, ?, ?)`, 
			t.inspectionId, cpu, memory, strings.Join(disks, ","), strings.Join(networks, ","),
		); err != nil {
			log.Error("db.Exec: ", err)
			return err
		}
	}

	return nil
}