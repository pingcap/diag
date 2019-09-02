package metric

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"os"
	"path"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
	"github.com/pingcap/tidb-foresight/analyzer/boot"
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
func (t *saveMetricTask) Run(c *boot.Config) *Metric {
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

	for _, file := range files {
		result := &queryResult{}
		data, err := ioutil.ReadFile(path.Join(metricDir, file.Name()))
		if err != nil {
			log.Error("read metric file:", err)
			return nil
		}
		if err := json.Unmarshal(data, &result); err != nil {
			log.Error("read metric content", err)
			return nil
		}
		if result.Status != "success" {
			log.Warnf("skip exceptional metric: %s", file.Name())
			continue
		}
		if result.Data.ResultType != "matrix" {
			continue
		}
		if err = t.insertMetricToInfluxdb(cli, c.InspectionId, result.Data.Result); err != nil {
			log.Error("insert metric to influxdb:", err)
			return nil
		}
	}

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
	step := 100
	count := len(matrix)
	for idx := 0; idx < count; idx += step {
		batch, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
			Database:  INFLUX_DB,
			Precision: "s",
		})
		if err != nil {
			return err
		}
		end := idx + step
		if end > count {
			end = count
		}
		for _, series := range matrix[idx:end] {
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
				p, err := influxdb.NewPoint(name, tags, fields, t)
				if err != nil {
					log.Error("insert a point to influxdb:", err)
					return err
				}
				batch.AddPoint(p)
			}
		}
		if err := cli.Write(batch); err != nil {
			return err
		}
	}
	return nil
}
