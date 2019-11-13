package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
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

type WrappedOptions struct {
	Options
}

func (w *WrappedOptions) GetTopology() (*model.Topology, error) {
	home := w.GetHome()
	instanceId := w.GetInstanceId()

	topo := model.Topology{}

	content, err := ioutil.ReadFile(path.Join(home, "topology", instanceId+".json"))
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

func (w *WrappedOptions) GetPrometheusEndpoint() (string, error) {
	topo, err := w.GetTopology()
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

func (w *WrappedOptions) GetTidbStatusEndpoints() ([]string, error) {
	endpoints := []string{}

	topo, err := w.GetTopology()
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

type Manager struct {
	opts Options
}

func New(opts Options) Collector {
	return &Manager{opts}
}

func (m *Manager) Collect() error {
	home := m.opts.GetHome()
	inspection := m.opts.GetInspectionId()
	start := time.Now()

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
	if cfg, err := json.Marshal(m.opts); err != nil {
		// if cannot, than panic.
		log.Error(err)
	} else {
		log.Info(
			fmt.Sprintf("Inspection %s collect with config: %s; and start from %s, ending in %s. Using time %s",
				inspection, string(cfg), start.String(), end.String(), end.Sub(start).String(),
			),
		)
	}

	return m.collectMeta(start, end)
}

func (m *Manager) collectTopology() error {
	home := m.opts.GetHome()
	instance := m.opts.GetInstanceId()
	inspection := m.opts.GetInspectionId()

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

func (m *Manager) collectArgs() error {
	home := m.opts.GetHome()
	inspection := m.opts.GetInspectionId()

	data, err := json.Marshal(m.opts)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(home, "inspection", inspection, "args.json"), data, os.ModePerm)
}

func (m *Manager) collectEnv() error {
	home := m.opts.GetHome()
	inspection := m.opts.GetInspectionId()

	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		env[pair[0]] = os.Getenv(pair[0])
	}

	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(home, "inspection", inspection, "env.json"), data, os.ModePerm)
}

func (m *Manager) collectMeta(start, end time.Time) error {
	home := m.opts.GetHome()
	inspection := m.opts.GetInspectionId()

	dict := map[string]time.Time{
		"create_time":  start,
		"inspect_time": start,
		"end_time":     end,
	}
	data, err := json.Marshal(dict)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(home, "inspection", inspection, "meta.json"), data, os.ModePerm)
}

func (m *Manager) collectRemote() error {
	status := make(map[string]error)
	for _, item := range m.opts.GetItems() {
		switch item {
		case "alert":
			status[item] = alert.New(&WrappedOptions{m.opts}).Collect()
		case "dmesg":
			status[item] = dmesg.New(&WrappedOptions{m.opts}).Collect()
		case "basic":
			status[item] = basic.New(&WrappedOptions{m.opts}).Collect()
		case "config":
			status[item] = config.New(&WrappedOptions{m.opts}).Collect()
		case "dbinfo":
			status[item] = dbinfo.New(&WrappedOptions{m.opts}).Collect()
		case "log":
			status[item] = logc.New(&WrappedOptions{m.opts}).Collect()
		case "metric":
			status[item] = metric.New(&WrappedOptions{m.opts}).Collect()
		case "profile":
			status[item] = profile.New(&WrappedOptions{m.opts}).Collect()
		case "network":
			status[item] = network.New(&WrappedOptions{m.opts}).Collect()
		}
	}

	return m.collectStatus(status)
}

func (m *Manager) collectStatus(status map[string]error) error {
	home := m.opts.GetHome()
	inspection := m.opts.GetInspectionId()
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
	return ioutil.WriteFile(path.Join(home, "inspection", inspection, "status.json"), data, os.ModePerm)
}
