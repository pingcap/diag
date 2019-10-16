package network

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"

	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type Options interface {
	GetHome() string
	GetInspectionId() string
	GetTopology() (*model.Topology, error)
}

type NetworkCollector struct {
	opts Options
}

func New(opts Options) *NetworkCollector {
	return &NetworkCollector{opts}
}

func (c *NetworkCollector) Collect() error {
	user, err := user.Current()
	if err != nil {
		return err
	}

	topo, err := c.opts.GetTopology()
	if err != nil {
		return err
	}

	for _, host := range topo.Hosts {
		if e := c.dmesg(user.Username, host.Ip); e != nil {
			if err == nil {
				err = e
			}
		}
	}

	return err
}

func (c *NetworkCollector) dmesg(user, ip string) error {
	p := path.Join(c.opts.GetHome(), "inspection", c.opts.GetInspectionId(), "net", ip)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path.Join(p, "netstat"))
	if err != nil {
		return err
	}
	defer f.Close()

	cmd := exec.Command(
		"ssh",
		fmt.Sprintf("%s@%s", user, ip),
		"netstat -s",
	)
	cmd.Stdout = f
	cmd.Stderr = os.Stderr

	log.Info(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Error("get network info:", err)
		return err
	}

	return nil
}
