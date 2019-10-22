package insight

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type parseInsightTask struct{}

func ParseInsight() *parseInsightTask {
	return &parseInsightTask{}
}

// Parse the output of insight collector
func (t *parseInsightTask) Run(c *boot.Config) *Insight {
	// 拿到一个初始化的 insight 对象
	insight := Insight{}

	ips, err := ioutil.ReadDir(path.Join(c.Src, "insight"))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("read dir:", err)
		}
		return nil
	}

	for _, ip := range ips {
		info, err := t.parse(path.Join(c.Src, "insight", ip.Name(), "collector.json"))
		if err != nil {
			log.Error("parse collect.json", err)
		}
		info.NodeIp = ip.Name()
		insight = append(insight, info)
	}

	return &insight
}

func (t *parseInsightTask) parse(fpath string) (*InsightInfo, error) {
	info := InsightInfo{}

	f, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}
