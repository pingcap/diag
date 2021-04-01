package profile

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"

	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type Options interface {
	GetHome() string
	InspectionID() string
}

type ProfileCollector struct {
	Options
}

func New(opts Options) *ProfileCollector {
	return &ProfileCollector{opts}
}

func (c *ProfileCollector) Collect() error {
	return nil
}

func (c *ProfileCollector) perfGolangProcess(name, ip, port string) {
	home := c.GetHome()
	inspection := c.InspectionID()

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
	home := c.GetHome()
	inspection := c.InspectionID()

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

	perf := exec.Command(
		"ssh",
		fmt.Sprintf("%s@%s", user.Username, ip),
		fmt.Sprintf("perf record -F 99 -g -p $(/usr/sbin/lsof -tiTCP:%s -sTCP:LISTEN -P -n) -o /tmp/perf.data -- sleep 60", port),
	)
	perf.Stdout = os.Stdout
	perf.Stderr = os.Stderr

	scp := exec.Command(
		"scp",
		fmt.Sprintf("%s@%s:/tmp/perf.data", user.Username, ip),
		path.Join(p, "perf.data"),
	)
	scp.Stdout = os.Stdout
	scp.Stderr = os.Stderr

	if err := utils.RunCommands(perf, scp); err != nil {
		log.Error("run remote perf:", err)
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
