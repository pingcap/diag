package alert

import (
	"encoding/json"
	"os"
	"path"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type parseAlertTask struct{}

func ParseAlert() *parseAlertTask {
	return &parseAlertTask{}
}

// Parse alert.json and return Alert, the alert.json looks like:
//  {
//	    "status": "success",
//		"data": {
//			"resultType": "vector",
//			"result": [{
//				"metric": {
//					"__name__": "ALERTS",
//					"alertname": "NODE_disk_read_latency_more_than_32ms",
//					"alertstate": "firing",
//					"device": "dm-2",
//					"env": "test-cluster",
//					"expr": "xxxxxx",
//					"instance": "172.16.4.246:9100",
//					"job": "overwritten-nodes",
//					"level": "warning"
//				},
//				"value": [1565182961.178, "1"]
//			}]
//		}
//	}
func (t *parseAlertTask) Run(c *boot.Config, m *boot.Model) *Alert {
	r := struct {
		Status string `json:"status"`
		Data   struct {
			Result Alert `json:"result"`
		} `json:"data"`
	}{}

	f, err := os.Open(path.Join(c.Src, "alert.json"))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("open file:", err)
		}
		return nil
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&r); err != nil {
		log.Error("decode:", err)
		m.InsertSymptom("exception", "parse alert.json failed", "contact developer")
		return nil
	}

	if r.Status != "success" {
		m.InsertSymptom("exception", "collect alert info failed", "check prometheus api")
		return nil
	}

	return &r.Data.Result
}
