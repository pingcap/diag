package task

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
)

type Dmesg []DmesgInfo

type DmesgInfo struct {
	Ip  string
	Log string
}

type ParseDmesgTask struct {
	BaseTask
}

func ParseDmesg(base BaseTask) Task {
	return &ParseDmesgTask{base}
}

func (t *ParseDmesgTask) Run() error {
	logs := Dmesg{}
	if !t.data.args.Collect(ITEM_BASIC) || t.data.status[ITEM_BASIC].Status != "success" {
		return nil
	}

	ips, err := ioutil.ReadDir(path.Join(t.src, "dmesg"))
	if err != nil {
		log.Error("read dir: ", err)
		return err
	}

	for _, ip := range ips {
		content, err := ioutil.ReadFile(path.Join(t.src, "dmesg", ip.Name(), "dmesg"))
		if err != nil {
			log.Error("read dmesg:", err)
		}
		logs = append(logs, DmesgInfo{
			Ip:  ip.Name(),
			Log: string(content),
		})
	}

	t.data.dmesg = logs

	return nil
}

type SaveDmesgTask struct {
	BaseTask
}

func SaveDmesg(base BaseTask) Task {
	return &SaveDmesgTask{base}
}

func (t *SaveDmesgTask) Run() error {
	if !t.data.args.Collect(ITEM_BASIC) || t.data.status[ITEM_BASIC].Status != "success" {
		return nil
	}

	for _, dmesg := range t.data.dmesg {
		if _, err := t.db.Exec(
			`INSERT INTO inspection_dmesg(inspection, node_ip, log) VALUES(?, ?, ?)`,
			t.inspectionId, dmesg.Ip, dmesg.Log,
		); err != nil {
			log.Error("db.Exec: ", err)
			return err
		}
	}

	return nil
}
