package task

import (
	"encoding/json"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
)

type AlertInfo []struct {
	Metric struct {
		Name string `json:"alertname"`
	} `json:"metric"`
	Value []interface{}
}

type ParseAlertTask struct {
	BaseTask
}

func ParseAlert(base BaseTask) Task {
	return &ParseAlertTask{base}
}

func (t *ParseAlertTask) Run() error {
	if !t.data.args.Collect(ITEM_METRIC) || t.data.status[ITEM_METRIC].Status != "success" {
		return nil
	}

	r := struct {
		Status string `json:"status"`
		Data   struct {
			Result AlertInfo `json:"result"`
		} `json:"data"`
	}{}

	f, err := os.Open(path.Join(t.src, "alert.json"))
	if err != nil {
		log.Error("open file: ", err)
		return err
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&r); err != nil {
		log.Error("decode: ", err)
		return t.SetStatus(ITEM_METRIC, "exception", "parse alert.json failed", "contact developer")
	}

	if r.Status != "success" {
		return t.SetStatus(ITEM_METRIC, "exception", "collect alert info failed", "check prometheus api")
	}

	t.data.alert = r.Data.Result

	return nil
}

type SaveAlertTask struct {
	BaseTask
}

func SaveAlert(base BaseTask) Task {
	return &SaveAlertTask{base}
}

func (t *SaveAlertTask) Run() error {
	if !t.data.args.Collect(ITEM_METRIC) || t.data.status[ITEM_METRIC].Status != "success" {
		return nil
	}

	for _, alert := range t.data.alert {
		if len(alert.Value) != 2 {
			continue
		}
		ts, ok := alert.Value[0].(float64)
		if !ok {
			log.Error("parse ts from alert failed")
			continue
		}
		v, ok := alert.Value[1].(string)
		if !ok {
			log.Error("parse value from alert failed")
			continue
		}
		if _, err := t.db.Exec(
			`INSERT INTO inspection_alerts(inspection, name, value, time) VALUES(?, ?, ?, ?)`,
			t.inspectionId, alert.Metric.Name, v, time.Unix(int64(ts), 0),
		); err != nil {
			log.Error("db.Exec: ", err)
			return err
		}
	}

	return nil
}
