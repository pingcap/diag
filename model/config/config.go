package config

import (
	"time"
)

// the config for a diagnosis or cluster.
// user can set auto schedule config for
// a cluster or set for each diagnosis.
type Config struct {
	// the instance this config belong to
	InstanceId string `json:"instance_id" gorm:"PRIMARY_KEY"`

	// should collector colecct hardware info
	CollectHardwareInfo bool `json:"collect_hardware_info"`

	// should collector collect software info
	CollectSoftwareInfo bool `json:"collect_software_info"`

	// should collector collect log
	CollectLog bool `json:"collect_log"`

	// should collector collect dmesg
	CollectDmesg bool `json:"collect_dmesg"`

	// auto schedule start time, eg. 00:00, 00:30, 01:00, 01:30
	AutoSchedStart string `json:"auto_sched_start"`

	// auto schedule day, eg. SUN, MON, TUE, WED, THU, FRI, SAT
	// if mutile ones specified, join them with comma.
	AutoSchedDay string `json:"auto_sched_day"`

	// auto schedule duration, minutes
	AutoSchedDuration int64 `json:"auto_sched_duration"`

	// when user click diagnose button, he will chose a time range
	// for collecting metric and log infomation.
	SchedRange []time.Time `json:"manual_sched_range"`

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

func (m *config) GetInstanceConfig(instanceId string) (*Config, error) {
	c := DefaultInstanceConfig(instanceId)

	if err := m.db.Where(&Config{InstanceId: instanceId}).FirstOrCreate(c).Error(); err != nil {
		return nil, err
	}

	return c, nil
}

func (m *config) ListInstanceConfig() ([]*Config, error) {
	configs := []*Config{}

	if err := m.db.Find(&configs).Error(); err != nil {
		return nil, err
	}

	return configs, nil
}

func (m *config) SetInstanceConfig(c *Config) error {
	return m.db.Save(c).Error()
}

func (m *config) DeleteInstanceConfig(instanceId string) error {
	return m.db.Delete(&Config{InstanceId: instanceId}).Error()
}
