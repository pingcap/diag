package basic

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"

	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type Options interface {
	GetHome() string
	GetInspectionId() string
	GetTopology() (*model.Topology, error)
}

type BasicCollector struct {
	opts Options
}

func New(opts Options) *BasicCollector {
	return &BasicCollector{opts}
}

func (b *BasicCollector) Collect() error {
	user, err := user.Current()
	if err != nil {
		return err
	}

	topo, err := b.opts.GetTopology()
	if err != nil {
		return err
	}

	for _, host := range topo.Hosts {
		if e := b.insight(user.Username, host.Ip); e != nil {
			if err == nil {
				err = e
			}
		}
	}

	return err
}

func (b *BasicCollector) insight(user, ip string) error {
	p := path.Join(b.opts.GetHome(), "inspection", b.opts.GetInspectionId(), "insight", ip)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path.Join(p, "collector.json"))
	if err != nil {
		return err
	}
	defer f.Close()

	install := exec.Command(
		"scp",
		fmt.Sprintf("%s@%s:%s", user, ip, path.Join(b.opts.GetHome(), "bin", "insight")),
		"/tmp/",
	)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	log.Info(install.Args)

	execute := exec.Command(
		"ssh",
		fmt.Sprintf("%s@%s", user, ip),
		fmt.Sprintf("bash -c \"%s\"", "sudo chmod 755 /tmp/insight && sudo /tmp/insight"),
	)
	execute.Stdout = f
	execute.Stderr = os.Stderr
	log.Info(execute.Args)

	if err := utils.StartCommands(install, execute); err != nil {
		log.Error("start remote insight:", err)
		return err
	}

	if err := utils.WaitCommands(install, execute); err != nil {
		log.Error("wait remote insight:", err)
		return err
	}

	return nil
}
