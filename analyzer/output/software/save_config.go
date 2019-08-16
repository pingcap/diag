package software

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type saveSoftwareConfigTask struct{}

func SaveSoftwareConfig() *saveSoftwareConfigTask {
	return &saveSoftwareConfigTask{}
}

// Save each component's config to database
func (t *saveSoftwareConfigTask) Run(m *boot.Model, c *boot.Config) {
	configDir := path.Join(c.Src, "config")

	configs, err := loadSoftwareConfigFiles(configDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("load software config:", err)
		}
		return
	}
	for _, cfg := range configs {
		if err := m.InsertInspectionConfigInfo(&model.ConfigInfo{
			InspectionId: c.InspectionId,
			NodeIp:       cfg.ip,
			Port:         strconv.Itoa(cfg.port),
			Component:    cfg.component,
			Config:       cfg.config,
		}); err != nil {
			log.Error("insert component config:", err)
			return
		}
	}
}

func loadSoftwareConfigFiles(dir string) ([]SoftwareConfig, error) {
	var configs []SoftwareConfig
	// "tikv" "172.16.5.7:20160" "tikv.toml"
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
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
