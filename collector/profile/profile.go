package profile

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"sync"

	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type Options interface {
	GetHome() string
	GetInspectionId() string
	GetTopology() (*model.Topology, error)
	GetComponents() []string
}

type ProfileCollector struct {
	opts Options
}

func New(opts Options) *ProfileCollector {
	return &ProfileCollector{opts}
}

func (c *ProfileCollector) Collect() error {
	var wg sync.WaitGroup

	topo, err := c.opts.GetTopology()
	if err != nil {
		return err
	}

	for _, host := range topo.Hosts {
		for _, comp := range host.Components {
			if c.shouldProfile(comp.Name, host.Ip, comp.Port) {
				wg.Add(1)
				go func() {
					switch comp.Name {
					case "pd":
						c.perfGolangProcess(comp.Name, host.Ip, comp.Port)
					case "tidb":
						c.perfGolangProcess(comp.Name, host.Ip, comp.StatusPort)
					case "tikv":
						c.perfRustProcess(comp.Name, host.Ip, comp.Port)
					}
					wg.Done()
				}()
			}
		}
	}

	return nil
}

func (c *ProfileCollector) shouldProfile(name, ip, port string) bool {
	comps := c.opts.GetComponents()

	if len(comps) == 0 {
		return true
	}

	for _, comp := range comps {
		if fmt.Sprintf("%s:%s:%s", name, ip, port) == comp {
			return true
		}
	}

	return false
}

func (c *ProfileCollector) perfGolangProcess(name, ip, port string) {
	home := c.opts.GetHome()
	inspection := c.opts.GetInspectionId()
	p := path.Join(home, "inspection", inspection, "profile", name, ip+":"+port)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		log.Error("create profile directory:", err)
		return
	}

	saveHttpResponse(fmt.Sprintf("http://%s:%s/debug/pprof/profile", ip, port), path.Join(p, "cpu.pb.gz"))
	saveHttpResponse(fmt.Sprintf("http://%s:%s/debug/pprof/heap", ip, port), path.Join(p, "mem.pb.gz"))
	saveHttpResponse(fmt.Sprintf("http://%s:%s/debug/pprof/block", ip, port), path.Join(p, "block.pb.gz"))
	saveHttpResponse(fmt.Sprintf("http://%s:%s/debug/pprof/goroutine", ip, port), path.Join(p, "goroutine.pb.gz"))
	saveHttpResponse(fmt.Sprintf("http://%s:%s/debug/pprof/mutex", ip, port), path.Join(p, "mutex.pb.gz"))
	saveHttpResponse(fmt.Sprintf("http://%s:%s/debug/pprof/threadcreate", ip, port), path.Join(p, "threadcreate.pb.gz"))
}

func (c *ProfileCollector) perfRustProcess(name, ip, port string) {
	home := c.opts.GetHome()
	inspection := c.opts.GetInspectionId()

	user, err := user.Current()
	if err != nil {
		log.Error("get user when perf rust process:", err)
		return
	}

	p := path.Join(home, "inspection", inspection, "profile", name, ip+":"+port)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		log.Error("create profile directory:", err)
		return
	}

	f, err := os.Create(path.Join(p, "perf.data"))
	if err != nil {
		log.Error("create perf.data:", err)
		return
	}
	defer f.Close()

	cmd := exec.Command(
		"ssh",
		fmt.Sprintf("%s@%s", user, ip),
		fmt.Sprintf("bash -c \"perf record -F 99 -g -p $(lsof -tiTCP:%s -sTCP:LISTEN -P -n) -o /dev/stdout -- sleep 60\"", port),
	)
	cmd.Stdout = f
	cmd.Stderr = os.Stderr

	log.Info(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Error("perf record:", err)
		return
	}
}

func saveHttpResponse(url, file string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Error("request profile:", err)
		return
	}
	defer resp.Body.Close()

	dst, err := os.Create(file)
	if err != nil {
		log.Error("create file:", err)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, resp.Body)
	if err != nil {
		log.Error("write file:", err)
		return
	}
}
