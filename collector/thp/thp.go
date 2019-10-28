/// This directory checks the transparent huge page for the instance application.

package thp

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/pingcap/tidb-foresight/model"
	"os"
	"os/exec"
	"os/user"
	"path"
	"regexp"

	log "github.com/sirupsen/logrus"
)

type Options interface {
	GetHome() string
	GetInspectionId() string
	GetTopology() (*model.Topology, error)
}

type TransparentHugePageCollector struct {
	opts Options
}

func New(opts Options) *TransparentHugePageCollector {
	return &TransparentHugePageCollector{opts}
}

type TransparentHugePage string

var thpMap map[string]TransparentHugePage

const (
	Always  TransparentHugePage = "always"
	MAdvise TransparentHugePage = "madvise"
	Never   TransparentHugePage = "never"
)

func init() {
	thpMap = make(map[string]TransparentHugePage)

	thpMap["always"] = Always
	thpMap["madvise"] = MAdvise
	thpMap["never"] = Never
}

func catchString(s string) (*TransparentHugePage, error) {
	re := regexp.MustCompile(`\[.+\]`)
	regResp := re.FindString(s)
	if len(regResp) == 0 {
		return nil, errors.New(fmt.Sprintf("cat failed, %s", s))
	}

	regResp = regResp[1 : len(regResp)-1]

	// must larger than 2
	if page, ok := thpMap[regResp]; !ok {
		return nil, errors.New(fmt.Sprintf("status %s not exists", page))
	} else {
		return &page, nil
	}
}

func (c *TransparentHugePageCollector) Collect() error {
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
			if e := c.collectThp(user.Username, host.Ip, comp.Port, comp.Name); e != nil {
				if err == nil {
					err = e
				}
			}
		}

	}

	return err
}

// TODO: this part of logic is paste from config/config, please find method to modify this.
func (c *TransparentHugePageCollector) collectThp(user, ip, port, comp string) error {
	if comp != "tidb" && comp != "pd" && comp != "tikv" {
		return nil
	}
	p := path.Join(c.opts.GetHome(), "thp", c.opts.GetInspectionId(), "config", comp, ip+":"+port)
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
		fmt.Sprintf("cat /sys/kernel/mm/redhat_transparent_huge"),
	)
	// parse buffer from thp
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Stderr = os.Stderr

	log.Info(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Error("collect config file:", err)
		return err
	}

	resp, err := catchString(out.String())
	if err != nil {
		return err
	}

	_, err = f.WriteString(string(*resp))
	if err != nil {
		return err
	}

	return nil
}
