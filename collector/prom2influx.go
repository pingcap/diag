// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

// This tool receives JSON metrics of Prometheus from stdin and writes them
// to a influxdb server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/prometheus/common/model"
)

type RebuildOptions struct {
	Host    string
	Port    int
	User    string
	Passwd  string
	DBName  string
	Cluster string // cluster name
	File    *os.File
	Chunk   int
}

type promResult struct {
	ResultType string
	Result     model.Matrix
}

type promDump struct {
	Status string
	Data   promResult
}

// queryDB convenience function to query the database
func queryDB(clnt influx.Client, db_name string, cmd string) (res []influx.Result, err error) {
	q := influx.Query{
		Command:  cmd,
		Database: db_name,
	}
	if response, err := clnt.Query(q); err == nil {
		if response.Error() != nil {
			return res, response.Error()
		}
		res = response.Results
	} else {
		return res, err
	}
	return res, nil
}

func slicePoints(data []*influx.Point, chunkSize int) [][]*influx.Point {
	var result [][]*influx.Point
	for i := 0; i < len(data); i += chunkSize {
		endPos := i + chunkSize
		if endPos > len(data) {
			endPos = len(data)
		}
		result = append(result, data[i:endPos])
	}
	return result
}

func newClient(opts *RebuildOptions) influx.Client {
	// connect to influxdb
	client, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     fmt.Sprintf("http://%s:%d", opts.Host, opts.Port),
		Username: opts.User,
		Password: opts.Passwd,
	})
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func buildPoints(
	series *model.SampleStream,
	client influx.Client,
	opts *RebuildOptions,
) []*influx.Point {
	var ptList []*influx.Point
	rawTags := series.Metric
	tags := make(map[string]string)
	for k, v := range rawTags {
		tags[string(k)] = string(v)
	}
	tags["cluster"] = opts.Cluster
	tags["monitor"] = "prometheus"
	measurement := tags["__name__"]
	for _, point := range series.Values {
		timestamp := point.Timestamp.Time()
		fields := map[string]interface{}{
			// model.SampleValue is alias of float64
			"value": float64(point.Value),
		}
		if pt, err := influx.NewPoint(measurement, tags, fields,
			timestamp); err == nil {
			ptList = append(ptList, pt)
		} // errored points are ignored
	}
	return ptList
}

func writeBatchPoints(data promDump, opts *RebuildOptions) error {
	for _, series := range data.Data.Result {
		client := newClient(opts)
		ptList := buildPoints(series, client, opts)

		for _, chunk := range slicePoints(ptList, opts.Chunk) {
			// create influx.Client and close it every time we write a BatchPoints
			// series to reduce memory usage on large dataset
			bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
				Database:  opts.DBName,
				Precision: "s",
			})
			if err != nil {
				return err
			}
			bp.AddPoints(chunk)
			// write batch points to influxdb
			if err := client.Write(bp); err != nil {
				return err
			}
		}
		client.Close()
	}
	return nil
}

func (opt *RebuildOptions) Load() error {
	// read JSON data from file
	input, err := io.ReadAll(opt.File)
	if err != nil {
		log.Fatal(err)
	}

	// decode JSON
	var data promDump
	if err = json.Unmarshal(input, &data); err != nil {
		log.Fatal(err)
	}

	// connect to influxdb
	client := newClient(opt)
	// create database has no side effect if database already exist
	_, err = queryDB(client, opt.DBName, fmt.Sprintf("CREATE DATABASE %s", opt.DBName))
	if err != nil {
		log.Fatal(err)
	}
	client.Close()

	if err := writeBatchPoints(data, opt); err != nil {
		log.Fatal(err)
	}
	return nil
}
