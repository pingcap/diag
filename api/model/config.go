package model

import (
	"database/sql"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	InstanceId            string `json:"instance_id"`
	CollectHardwareInfo   bool   `json:"collect_hardware_info"`
	CollectSoftwareInfo   bool   `json:"collect_software_info"`
	CollectLog            bool   `json:"collect_log"`
	CollectLogDuration    int    `json:"collect_log_duration"`
	CollectMetricDuration int    `json:"collect_metric_duration"`
	CollectDemsg          bool   `json:"collect_demsg"`
	AutoSchedDuration     string `json:"auto_sched_duration"`
	AutoSchedStart        string `json:"auto_sched_start"`
	ReportKeepDuration    string `json:"report_keep_duration"`
}

func DefaultInstanceConfig(instanceId string) *Config {
	return &Config{
		InstanceId: instanceId,
	}
}

func (m *Model) GetInstanceConfig(instanceId string) (*Config, error) {
	config := &Config{}

	row := m.db.QueryRow(
		"SELECT instance,c_hardw,c_softw,c_log,c_log_d,c_metric_d,c_demsg,s_cron,r_duration FROM configs WHERE instance = ?",
		instanceId,
	)
	var cHardw, cSoftw, cLog, cDemsg int
	var sCron string
	err := row.Scan(
		&config.InstanceId, &cHardw, &cSoftw, &cLog, &config.CollectLogDuration,
		&config.CollectMetricDuration, &cDemsg, &sCron, &config.ReportKeepDuration,
	)
	ss := strings.Split(sCron, "/")
	if err == nil {
		config.CollectHardwareInfo = cHardw != 0
		config.CollectSoftwareInfo = cSoftw != 0
		config.CollectLog = cLog != 0
		config.CollectDemsg = cDemsg != 0
		if len(ss) > 0 {
			config.AutoSchedStart = ss[0]
		}
		if len(ss) > 1 {
			config.AutoSchedDuration = ss[1]
		}
		return config, nil
	} else if err == sql.ErrNoRows {
		log.Error("no config for instance ", instanceId, ": ", err)
		return nil, nil
	} else {
		log.Error("error query instance config: ", err)
		return nil, err
	}
}

func (m *Model) SetInstanceConfig(config *Config) error {
	toInt := func(b bool) int {
		if b {
			return 1
		} else {
			return 0
		}
	}
	_, err := m.db.Exec(
		`REPLACE INTO configs(instance,c_hardw,c_softw,c_log,c_log_d,c_metric_d,c_demsg,s_cron,r_duration)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		config.InstanceId, toInt(config.CollectHardwareInfo), toInt(config.CollectSoftwareInfo),
		toInt(config.CollectLog), config.CollectLogDuration, config.CollectMetricDuration, toInt(config.CollectDemsg),
		config.AutoSchedStart + "/" + config.AutoSchedDuration, config.ReportKeepDuration,
	)

	return err
}

func (m *Model) DeleteInstanceConfig(instanceId string) error {
	_, err := m.db.Exec("DELETE FROM configs where instance = ?", instanceId)
	if err != nil {
		log.Error("db.Exec: ", err)
		return err
	} else {
		return nil
	}
}
