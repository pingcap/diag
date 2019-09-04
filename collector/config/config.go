package config

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

type ConfigCollector struct {
	opts Options
}

func New(opts Options) *ConfigCollector {
	return &ConfigCollector{opts}
}

func (c *ConfigCollector) Collect() error {
	user, err := user.Current()
	if err != nil {
		return err
	}

	topo, err := c.opts.GetTopology()
	if err != nil {
		return err
	}

	for _, host := range topo.Hosts {
		for _, comp := range host.Components {
			if e := c.config(user.Username, host.Ip, comp.Port, comp.Name, comp.DeployDir); e != nil {
				if err == nil {
					err = e
				}
			}
		}

	}

	return err
}

func (c *ConfigCollector) config(user, ip, port, comp, depdir string) error {
	if comp != "tidb" && comp != "pd" && comp != "tikv" {
		return nil
	}
	p := path.Join(c.opts.GetHome(), "inspection", c.opts.GetInspectionId(), "config", comp, ip+":"+port)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path.Join(p, comp+".toml"))
	if err != nil {
		return err
	}
	defer f.Close()

	cmd := exec.Command(
		"ssh",
		fmt.Sprintf("%s@%s", user, ip),
		fmt.Sprintf("bash -c \"cat %s/conf/%s.toml\"", depdir, comp),
	)
	cmd.Stdout = f
	cmd.Stderr = os.Stderr

	log.Info(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Error("collect config file:", err)
		return err
	}

	return nil
}
