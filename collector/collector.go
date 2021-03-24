package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pingcap/tidb-foresight/collector/alert"
	"github.com/pingcap/tidb-foresight/collector/basic"
	"github.com/pingcap/tidb-foresight/collector/config"
	"github.com/pingcap/tidb-foresight/collector/dbinfo"
	"github.com/pingcap/tidb-foresight/collector/dmesg"
	logc "github.com/pingcap/tidb-foresight/collector/log"
	"github.com/pingcap/tidb-foresight/collector/metric"
	"github.com/pingcap/tidb-foresight/collector/network"
	"github.com/pingcap/tidb-foresight/collector/profile"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/wrapper/db"
	log "github.com/sirupsen/logrus"
)

type Collector interface {
	Collect() error
}

type Options interface {
	GetInstanceId() string
	GetInspectionId() string
	GetHome() string
	GetItems() []string
	GetScrapeBegin() (time.Time, error)
	GetScrapeEnd() (time.Time, error)
	GetComponents() []string
}

type Manager struct {
	Options
}

func New(opts Options) Collector {
	return &Manager{opts}
}

func (m *Manager) Collect() error {
	home := m.GetHome()
	inspection := m.GetInspectionId()
	start := time.Now()

	// mkdir for collection results.
	if err := os.MkdirAll(path.Join(home, "inspection", inspection), os.ModePerm); err != nil {
		return err
	}

	if err := m.collectTopology(); err != nil {
		return err
	}
	if err := m.collectArgs(); err != nil {
		return err
	}
	if err := m.collectEnv(); err != nil {
		return err
	}
	if err := m.collectRemote(); err != nil {
		return err
	}

	end := time.Now()
	if cfg, err := json.Marshal(m.Options); err != nil {
		// if cannot, than panic.
		log.Error(err)
	} else {
		log.Infof(
			"Inspection %s collect with config: %s; and start from %s, ending in %s. Using time %s",
			inspection, string(cfg), start.String(), end.String(), end.Sub(start).String(),
		)

	}

	return m.collectMeta(start, end)
}

// collectTopology runs in local machine.
// It move the topology file from topology/{instance_id}.json to inspection/{topology}.json
func (m *Manager) collectTopology() error {
	home := m.GetHome()
	instance := m.GetInstanceId()
	inspection := m.GetInspectionId()

	m.GetModel().UpdateInspectionMessage(inspection, "collecting topology...")

	src, err := os.Open(path.Join(home, "topology", instance+".json"))
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(path.Join(home, "inspection", inspection, "topology.json"))
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// collectArgs runs in local machine.
// It generate an args.json by it's opts.
func (m *Manager) collectArgs() error {
	home := m.GetHome()
	inspection := m.GetInspectionId()

	m.GetModel().UpdateInspectionMessage(inspection, "collecting args...")

	data, err := json.Marshal(m.Options)
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(home, "inspection", inspection, "args.json"), data, os.ModePerm)
}

// collectArgs runs in local machine.
// It generate an args.json by it's environment variables.
func (m *Manager) collectEnv() error {
	home := m.GetHome()
	inspection := m.GetInspectionId()

	m.GetModel().UpdateInspectionMessage(inspection, "collecting env...")

	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		env[pair[0]] = os.Getenv(pair[0])
	}

	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(home, "inspection", inspection, "env.json"), data, os.ModePerm)
}

func (m *Manager) collectMeta(start, end time.Time) error {
	home := m.GetHome()
	inspection := m.GetInspectionId()

	m.GetModel().UpdateInspectionMessage(inspection, "collecting meta...")

	dict := map[string]time.Time{
		"create_time":  start,
		"inspect_time": start,
		"end_time":     end,
	}
	data, err := json.Marshal(dict)
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(home, "inspection", inspection, "meta.json"), data, os.ModePerm)
}

func (m *Manager) collectRemote() error {
	// mutex is using to protect status.
	var wg sync.WaitGroup
	var statusMutex sync.Mutex
	status := make(map[string]error)

	// build arrays for collector.
	toCollectMap := make(map[string]Collector, 0)

	for _, item := range m.GetItems() {
		switch item {
		case "alert":
			toCollectMap[item] = alert.New(m)
		case "dmesg":
			toCollectMap[item] = dmesg.New(m)
		case "basic":
			toCollectMap[item] = basic.New(m)
		case "config":
			toCollectMap[item] = config.New(m)
		case "dbinfo":
			toCollectMap[item] = dbinfo.New(m)
		case "log":
			toCollectMap[item] = logc.New(m)
		case "metric":
			toCollectMap[item] = metric.New(m)
		case "profile":
			toCollectMap[item] = profile.New(m)
		case "network":
			toCollectMap[item] = network.New(m)
		}
	}

	inspId := m.GetInspectionId()
	for item, collector := range toCollectMap {
		wg.Add(1)
		go func(innerCollector Collector, key string) {
			defer wg.Done()
			collected := innerCollector.Collect()
			log.Info(fmt.Sprintf("[Collector] Inspection %s collect %s done.", inspId, item))
			statusMutex.Lock()
			defer statusMutex.Unlock()
			status[key] = collected
		}(collector, item)
	}

	wg.Wait()
	return m.collectStatus(status)
}

func (m *Manager) collectStatus(status map[string]error) error {
	home := m.GetHome()
	inspection := m.GetInspectionId()

	m.GetModel().UpdateInspectionMessage(inspection, "collecting status...")

	dict := make(map[string]map[string]string)
	for item, err := range status {
		if err == nil {
			dict[item] = map[string]string{
				"status": "success",
			}
		} else {
			dict[item] = map[string]string{
				"status":  "error",
				"message": err.Error(),
			}
		}
	}

	data, err := json.Marshal(dict)
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(home, "inspection", inspection, "status.json"), data, os.ModePerm)
}

func (m *Manager) GetTopology() (*model.Topology, error) {
	home := m.GetHome()
	instanceId := m.GetInstanceId()

	topo := model.Topology{}

	content, err := os.ReadFile(path.Join(home, "topology", instanceId+".json"))
	if err != nil {
		log.Error("read file:", err)
		return nil, err
	}

	if err = json.Unmarshal(content, &topo); err != nil {
		log.Error("unmarshal:", err)
		return nil, err
	}

	return &topo, nil
}

func (m *Manager) GetPrometheusEndpoint() (string, error) {
	topo, err := m.GetTopology()
	if err != nil {
		return "", err
	}

	for _, host := range topo.Hosts {
		for _, comp := range host.Components {
			if comp.Name == "prometheus" {
				return host.Ip + ":" + comp.Port, nil
			}
		}
	}

	return "", errors.New("component prometheus not found")
}

func (m *Manager) GetTidbStatusEndpoints() ([]string, error) {
	endpoints := []string{}

	topo, err := m.GetTopology()
	if err != nil {
		return endpoints, err
	}

	for _, host := range topo.Hosts {
		for _, comp := range host.Components {
			if comp.Name == "tidb" {
				endpoints = append(endpoints, host.Ip+":"+comp.StatusPort)
			}
		}
	}

	return endpoints, nil
}

func (m *Manager) GetModel() model.Model {
	db, err := db.Open(path.Join(m.GetHome(), "sqlite.db"))
	if err != nil {
		log.Panic(err)
	}

	return model.New(db)
}
