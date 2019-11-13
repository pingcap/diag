package basic

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"

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
		ports := []string{}
		for _, comp := range host.Components {
			ports = append(ports, comp.Port)
		}
		if e := b.insight(user.Username, host.Ip, ports); e != nil {
			if err == nil {
				err = e
			}
		}
	}

	return err
}

func (b *BasicCollector) insight(user, ip string, ports []string) error {
	p := path.Join(b.opts.GetHome(), "inspection", b.opts.GetInspectionId(), "insight", ip)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path.Join(p, "collector.json"))
	if err != nil {
		return err
	}
	defer f.Close()

	clean := exec.Command(
		"ssh",
		fmt.Sprintf("%s@%s", user, ip),
		"sudo rm -f /tmp/insight",
	)
	clean.Stdout = os.Stdout
	clean.Stderr = os.Stderr
	log.Info(clean.Args)

	install := exec.Command(
		"scp",
		path.Join(b.opts.GetHome(), "bin", "insight"),
		fmt.Sprintf("%s@%s:/tmp/", user, ip),
	)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr

	execute := exec.Command(
		"ssh",
		fmt.Sprintf("%s@%s", user, ip),
		fmt.Sprintf("sudo chmod 755 /tmp/insight && sudo /tmp/insight --port %s", strings.Join(ports, ",")),
	)
	execute.Stdout = f
	execute.Stderr = os.Stderr

	if err := utils.RunCommands(clean, install, execute); err != nil {
		log.Error("run remote insight:", err)
		return err
	}

	return nil
}
