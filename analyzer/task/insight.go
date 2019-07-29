package task

import (
	"os"
	"path"
	"io/ioutil"
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

type Insight []*InsightInfo

type InsightInfo struct {
	NodeIp string
	Info struct {
		Meta struct {
			Tidb []struct{
				Version string `json:"release_version"`
			} `json:"tidb"`
			Tikv []struct{
				Version string `json:"release_version"`
			} `json:"tikv"`
			Pd []struct{
				Version string `json:"release_version"`
			} `json:"pd"`
		} `json:"meta"`
	}
	Sysinfo struct {
		Os struct {
			Name string `json:"name"`
		} `json:"os"`
		Kernel struct {
			Release string `json:"release"`
		}
		Cpu struct {
			Model string `json:"model"`
		} `json:"cpu"`
		Memory struct {
			Type string `json:"type"`
			Speed int `json:"speed"`
			Size int `json:"size"`
		} `json:"memory"`
		Storage []struct {
			Name string `json:"name"`
		} `json:"storage"`
		Network []struct {
			Name string `json:"name"`
			Speed int `json:"speed"`
		} `json:"network"`
		Ntp struct {
			Sync string `json:"sync"`
			Offset float64 `json:"offset"`
			Status string `json:"status"`
		} `json:"ntp"`
		Partitions []struct {
			Name string `json:"name"`
			Subdev []struct {
				Name string `json:"name"`
			} `json:"subdev"`
		} `json:"partitions"`
	} `json:"sysinfo"`
}

type ParseInsightTask struct {
	BaseTask
}

func ParseInsight(base BaseTask) Task {
	return &ParseInsightTask {base}
}

func (t *ParseInsightTask) Run() error {
	insight := Insight{}

	if !t.data.args.Collect(ITEM_BASIC) || t.data.status[ITEM_BASIC].Status != "success" {
		return nil
	}

	ips, err := ioutil.ReadDir(path.Join(t.src, "insight"))
	if err != nil {
		log.Error("read dir: ", err)
		return err
	}

	for _, ip := range ips {
		info, err := t.parse(path.Join(t.src, "insight", ip.Name(), "collector.json"))
		if err != nil {
			log.Error("parse collect.json", err)
		}
		insight = append(insight, info)
	}

	t.data.insight = insight

	return nil
}

func (t *ParseInsightTask) parse(fpath string) (*InsightInfo, error) {
	info := InsightInfo{}

	f, err := os.Open(fpath)
	if err != nil {
		log.Error("open file: ", err)
		return nil, err
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&info); err != nil {
		log.Error("decode: ", err)
		return nil, err
	}

	return &info, nil
}