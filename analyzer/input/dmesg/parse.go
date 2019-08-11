package dmesg

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type parseDmesgTask struct{}

func ParseDmesg() *parseDmesgTask {
	return &parseDmesgTask{}
}

// Read every dmesg file collected by collector, they are the same with dmesg command output
func (t *parseDmesgTask) Run(c *boot.Config) *Dmesg {
	logs := Dmesg{}

	ips, err := ioutil.ReadDir(path.Join(c.Src, "dmesg"))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("read dir:", err)
		}
		return nil
	}

	for _, ip := range ips {
		content, err := ioutil.ReadFile(path.Join(c.Src, "dmesg", ip.Name(), "dmesg"))
		if err != nil {
			log.Error("read dmesg:", err)
			continue
		}
		logs = append(logs, DmesgInfo{
			Ip:  ip.Name(),
			Log: string(content),
		})
	}

	return &logs
}
