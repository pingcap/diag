package args

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"strings"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

// Check if a specified item is choised to collect by user
func (a *Args) Collect(iname string) bool {
	items := strings.Split(a.Collects, ",")
	for _, item := range items {
		if iname == strings.Split(item, ":")[0] {
			return true
		}
	}
	return false
}

type parseArgsTask struct{}

func ParseArgs() *parseArgsTask {
	return &parseArgsTask{}
}

// Parse args.json and return Args, the args.json looks like:
//	{
//		"begin": "2019-08-07T08:52:39-04:00",
//		"end": "2019-08-07T09:02:39-04:00",
//		"log_spliter": "/home/tidb/tidb-foresight/bin/spliter",
//		"instance_id": "6478c40b-0190-49f8-b0e8-641016d5ff2f",
//		"collect": "metric,basic,dbinfo,config,profile",
//		"inspection_id": "0e56ba0d-c1c9-4c79-85d1-eceb832708d4",
//		"data_dir": "/home/tidb/tidb-foresight/inspection",
//		"topology": "/home/tidb/tidb-foresight/topology/6478c40b-0190-49f8-b0e8-641016d5ff2f.json"
//	}
func (t *parseArgsTask) Run(c *boot.Config, m *boot.Model) *Args {
	content, err := ioutil.ReadFile(path.Join(c.Src, "args.json"))
	if err != nil {
		log.Error("read file:", err)
		m.InsertSymptom("exception", "parse args.json", "contact developer")
		return nil
	}

	args := &Args{}
	if err = json.Unmarshal(content, args); err != nil {
		log.Error("unmarshal:", err)
		m.InsertSymptom("exception", "parse args.json", "contact developer")
		return nil
	}

	return args
}
