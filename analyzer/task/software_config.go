package task

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type SaveSoftwareConfigTask struct {
	BaseTask
}

func SaveSoftwareConfig(base BaseTask) Task {
	return &SaveSoftwareConfigTask{base}
}

func (t *SaveSoftwareConfigTask) Run() error {
	if !t.data.args.Collect(ITEM_CONFIG) || t.data.status[ITEM_CONFIG].Status != "success" {
		return nil
	}
	configDir := path.Join(t.src, "config")

	configs, err := loadSoftwareConfigFiles(configDir)
	if err != nil {
		return err
	}
	for _, c := range configs {
		if _, err := t.db.Exec(
			`INSERT INTO software_config(inspection, node_ip, port, component, config) VALUES(?, ?, ?, ?, ?)`,
			t.inspectionId, c.ip, c.port, c.component, c.config,
		); err != nil {
			log.Error("db.Exec: ", err)
			return err
		}
	}

	return nil
}

type SoftwareConfig struct {
	ip        string
	port      int
	component string
	config    string
}

func loadSoftwareConfigFiles(dir string) ([]SoftwareConfig, error) {
	var configs []SoftwareConfig
	// "tikv" "172.16.5.7:20160" "tikv.toml"
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		filename := info.Name()
		if filepath.Ext(filename) != ".toml" {
			return nil
		}
		paths := strings.Split(path, string(filepath.Separator))
		if len(paths) < 4 {
			return fmt.Errorf("wrong slow query log file path: %v\n", paths)
		}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		configStr := string(b)
		ip, port, err := parseIPAndPort(paths[len(paths)-2])
		if err != nil {
			return err
		}
		config := SoftwareConfig{
			ip:        ip,
			port:      port,
			component: paths[len(paths)-3],
			config:    configStr,
		}
		configs = append(configs, config)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return configs, nil
}

func parseIPAndPort(raw string) (string, int, error) {
	s := strings.Split(raw, ":")
	ip := s[0]
	port, err := strconv.Atoi(s[1])
	if err != nil {
		return "", 0, err
	}
	return ip, port, nil
}
