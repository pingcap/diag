package model

import (
	"database/sql"
	"time"

	log "github.com/sirupsen/logrus"
)

// the config for a diagnosis or cluster.
// user can set auto schedule config for
// a cluster or set for each diagnosis.
type Config struct {
	// the instance this config belong to
	InstanceId string `json:"instance_id"`

	// should collector colecct hardware info
	CollectHardwareInfo bool `json:"collect_hardware_info"`

	// should collector collect software info
	CollectSoftwareInfo bool `json:"collect_software_info"`

	// should collector collect log
	CollectLog bool `json:"collect_log"`

	// should collector collect dmesg
	CollectDemsg bool `json:"collect_demsg"`

	// auto schedule start time, eg. 00:00, 00:30, 01:00, 01:30
	AutoSchedStart string `json:"auto_sched_start"`

	// auto schedule duration, minutes
	AutoSchedDuration int64 `json:"auto_sched_duration"`

	// when user click diagnose button, he will chose a time range
	// for collecting metric and log infomation.
	ManualSchedRange []time.Time `json:"manual_sched_range"`

	// how long before the foresight gc remove a report
	ReportKeepDuration int64 `json:"report_keep_duration"`
}

// the default config for a instance on it's born
func DefaultInstanceConfig(instanceId string) *Config {
	return &Config{
		InstanceId:        instanceId,
		AutoSchedStart:    "00:00",
		AutoSchedDuration: 60,
	}
}

func (m *Model) GetInstanceConfig(instanceId string) (*Config, error) {
	config := &Config{}

	row := m.db.QueryRow(
		"SELECT instance,c_hardw,c_softw,c_log,c_demsg,s_start,s_duration,r_duration FROM configs WHERE instance = ?",
		instanceId,
	)
	var cHardw, cSoftw, cLog, cDemsg int64
	if err := row.Scan(
		&config.InstanceId, &cHardw, &cSoftw, &cLog, &cDemsg,
		&config.AutoSchedStart, &config.AutoSchedDuration, &config.ReportKeepDuration,
	); err == nil {
		config.CollectHardwareInfo = cHardw != 0
		config.CollectSoftwareInfo = cSoftw != 0
		config.CollectLog = cLog != 0
		config.CollectDemsg = cDemsg != 0
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
		`REPLACE INTO configs(instance,c_hardw,c_softw,c_log,c_demsg,s_start,s_duration,r_duration)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		config.InstanceId, toInt(config.CollectHardwareInfo), toInt(config.CollectSoftwareInfo),
		toInt(config.CollectLog), toInt(config.CollectDemsg), config.AutoSchedStart, config.AutoSchedDuration,
		config.ReportKeepDuration,
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
