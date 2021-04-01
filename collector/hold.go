package collector

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	logc "github.com/pingcap/tidb-foresight/collector/log"
	"github.com/pingcap/tidb-foresight/collector/profile"
	log "github.com/sirupsen/logrus"
)

type Collector interface {
	Collect() error
}

type Options interface {
	ClusterName() string
	InspectionID() string
	GetHome() string
	GetItems() []string
	GetScrapeBegin() (time.Time, error)
	GetScrapeEnd() (time.Time, error)
}

type CManager struct {
	Options
}

func New(opts Options) Collector {
	return &CManager{opts}
}

func (m *CManager) Collect() error {
	home := m.GetHome()
	inspection := m.InspectionID()
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
func (m *CManager) collectTopology() error {
	home := m.GetHome()
	instance := m.InspectionID()
	inspection := m.InspectionID()

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
func (m *CManager) collectArgs() error {
	home := m.GetHome()
	inspection := m.InspectionID()

	data, err := json.Marshal(m.Options)
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(home, "inspection", inspection, "args.json"), data, os.ModePerm)
}

// collectArgs runs in local machine.
// It generate an args.json by it's environment variables.
func (m *CManager) collectEnv() error {
	home := m.GetHome()
	inspection := m.InspectionID()

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

func (m *CManager) collectMeta(start, end time.Time) error {
	home := m.GetHome()
	inspection := m.InspectionID()

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

func (m *CManager) collectRemote() error {
	// mutex is using to protect status.
	var wg sync.WaitGroup
	var statusMutex sync.Mutex
	status := make(map[string]error)

	// build arrays for collector.
	toCollectMap := make(map[string]Collector, 0)

	for _, item := range m.GetItems() {
		switch item {
		//case "alert":
		//	toCollectMap[item] = alert.New(m)
		//case "dbinfo":
		//	toCollectMap[item] = dbinfo.New(m)
		case "log":
			toCollectMap[item] = logc.New(m)
		case "profile":
			toCollectMap[item] = profile.New(m)
		}
	}

	inspId := m.InspectionID()
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

func (m *CManager) collectStatus(status map[string]error) error {
	home := m.GetHome()
	inspection := m.InspectionID()

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
