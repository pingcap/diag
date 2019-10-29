package metric

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path"
	"runtime"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

const INFLUX_DB = "inspection"

// queryResult is the result returned from the prometheus api
type queryResult struct {
	Status string  `json:"status"`
	Data   metricT `json:"data"`
}

// ParseMetricTask parses the metric from json files
type saveMetricTask struct{}

// ParseMetric builds and return the saveMetricTask
func SaveMetric() *saveMetricTask {
	return &saveMetricTask{}
}

// Run the task which parses all the metric files collected
// by a metric collector
func (t *saveMetricTask) Run(c *boot.Config, m *boot.Model) *Metric {
	metricDir := path.Join(c.Src, "metric")
	files, err := ioutil.ReadDir(metricDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("load metric dir:", err)
		}
		return nil
	}

	cli, err := t.initInfluxdbClient()
	if err != nil {
		log.Error("connect influxdb:", err)
		return nil
	}
	defer cli.Close()

	tl := utils.NewTokenLimiter(uint(runtime.NumCPU()))
	start := time.Now()
	rand.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
	for idx, file := range files {
		file := file
		go func(tok *utils.Token) {
			defer tl.Put(tok)
			result := &queryResult{}
			data, err := ioutil.ReadFile(path.Join(metricDir, file.Name()))
			if err != nil {
				log.Error("read metric file:", err)
				return
			}
			if err := json.Unmarshal(data, &result); err != nil {
				log.Errorf("read metric content(%s): %s", file.Name(), err)
				return
			}
			if result.Status != "success" {
				log.Warnf("skip exceptional metric: %s", file.Name())
				return
			}
			if result.Data.ResultType != "matrix" {
				return
			}
			if err = t.insertMetricToInfluxdb(cli, c.InspectionId, result.Data.Result); err != nil {
				log.Error("insert metric to influxdb:", err)
			}
		}(tl.Get())
		elapsed := int(time.Now().Sub(start).Seconds())
		left := int32(elapsed*10000/(idx+1)*(len(files)-idx-1)) / 10000
		if err := m.UpdateInspectionEstimateLeftSec(c.InspectionId, left); err != nil {
			log.Error("update estimate left sec:", err)
		}
	}
	tl.Wait()

	// Return an empty struct, which will be used to fill other tasks' input args to make
	// sure this task execute before them.
	return &Metric{}
}

func (t *saveMetricTask) initInfluxdbClient() (influxdb.Client, error) {
	addr := os.Getenv("INFLUX_ADDR")
	if addr == "" {
		addr = "http://127.0.0.1:9528"
	}
	cli, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     addr,
		Username: os.Getenv("INFLUX_USER"),
		Password: os.Getenv("INFLUX_PWD"),
	})
	if err != nil {
		return nil, err
	}

	resp, err := cli.Query(influxdb.NewQuery("CREATE database "+INFLUX_DB, "", ""))
	if err != nil {
		cli.Close()
		return nil, err
	}
	if resp.Error() != nil {
		cli.Close()
		return nil, resp.Error()
	}

	return cli, nil
}

func (t *saveMetricTask) insertMetricToInfluxdb(cli influxdb.Client, inspectionId string, matrix matrixT) error {
	// Use a batch method to improve the speed to import
	batchWriter := NewBatchWriter(cli, 10000)
	defer batchWriter.Close()
	for _, series := range matrix {
		tags := series.Metric

		name, ok := tags["__name__"]
		if !ok {
			continue
		}
		tags["inspectionid"] = inspectionId

		for _, point := range series.Points {
			if math.IsNaN(point.V) || math.IsInf(point.V, 0) {
				continue
			}
			fields := map[string]interface{}{
				"value": point.V,
			}
			t := time.Unix(point.T, 0)
			if p, err := influxdb.NewPoint(name, tags, fields, t); err != nil {
				return err
			} else if err := batchWriter.Write(INFLUX_DB, p); err != nil {
				return err
			}
		}
	}
	return nil
}
