package task

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"os"
	"path"
	"strconv"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"
)

/*
 * The concept and definion of Matrix, Series, and Point comes frome prometheus
 */

// Matrix is a slice of Seriess that implements sort.Interface and
// has a String method.
type Matrix []Series

// Series is a stream of data points belonging to a metric.
type Series struct {
	Metric map[string]string `json:"metric"`
	Points []Point           `json:"values"`
}

// Point represents a single data point for a given timestamp.
type Point struct {
	T int64
	V float64
}

// UnmarshalJSON unmarshals data to a point
func (p *Point) UnmarshalJSON(data []byte) error {
	var a [2]interface{}
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	p.T = int64(a[0].(float64))
	val, err := strconv.ParseFloat(a[1].(string), 64)
	if err != nil {
		return err
	}
	p.V = val

	return nil
}

// Metric represents a metric returned from the prometheus api
type Metric struct {
	ResultType string `json:"resultType"`
	Result     Matrix `json:"result"`
}

// QueryResult is the result returned from the prometheus api
type QueryResult struct {
	Status string `json:"status"`
	Data   Metric `json:"data"`
}

// ParseMetricTask parses the metric from json files
type ParseMetricTask struct {
	BaseTask
}

// Run the task which parses all the metric files collected
// by a metric collector
func (t *ParseMetricTask) Run() error {
	if !t.data.args.Collect(ITEM_METRIC) || t.data.status[ITEM_METRIC].Status != "success" {
		return nil
	}

	metricDir := path.Join(t.src, "metric")
	files, err := ioutil.ReadDir(metricDir)
	if err != nil {
		return err
	}
	var matrix Matrix
	for _, file := range files {
		result := &QueryResult{}
		data, err := ioutil.ReadFile(path.Join(metricDir, file.Name()))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &result); err != nil {
			return err
		}
		if result.Status != "success" {
			log.Warnf("skip exceptional metric: %s", file.Name())
			continue
		}
		if result.Data.ResultType != "matrix" {
			continue
		}
		matrix = append(matrix, result.Data.Result...)
	}

	t.data.matrix = matrix
	return nil
}

// ParseMetric builds and return the ParseMetricTask
func ParseMetric(base BaseTask) Task {
	return &ParseMetricTask{base}
}

// SaveMetricTask saves the metrics into influxdb
type SaveMetricTask struct {
	BaseTask
}

// Run a task to save metrics into influxdb
func (t *SaveMetricTask) Run() error {
	if !t.data.args.Collect(ITEM_METRIC) || t.data.status[ITEM_METRIC].Status != "success" {
		return nil
	}

	database := "inspection"
	cli, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr:     "http://127.0.0.1:8086",
		Username: os.Getenv("INFLUX_USER"),
		Password: os.Getenv("INFLUX_PWD"),
	})
	if err != nil {
		return err
	}
	defer cli.Close()
	resp, err := cli.Query(influxdb.NewQuery(
		"CREATE DATABASE "+database, "", ""))
	if err != nil {
		return err
	}
	if resp.Error() != nil {
		return resp.Error()
	}

	// Use a batch method to improve the speed to import
	step := 100
	count := len(t.data.matrix)
	for idx := 0; idx < count; idx += step {
		batch, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
			Database:  database,
			Precision: "s",
		})
		if err != nil {
			return err
		}
		end := idx + step
		if end > count {
			end = count
		}
		for _, series := range t.data.matrix[idx:end] {
			tags := series.Metric

			name, ok := tags["__name__"]
			if !ok {
				continue
			}
			tags["inspectionid"] = t.inspectionId

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

// SaveMetric returns an instance of SaveMetricTask
func SaveMetric(base BaseTask) Task {
	return &SaveMetricTask{base}
}
