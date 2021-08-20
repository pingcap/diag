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
	"io/fs"
	"log"
	"os"
	"path"
	"strings"

	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/pingcap/diag/utils"
	"github.com/pingcap/tiup/pkg/tui/progress"
	"github.com/prometheus/common/model"
)

func (opt *RebuildOptions) LoadMetrics() error {
	f, err := os.Open(opt.File)
	if err != nil {
		return err
	}

	var input []byte
	var readErr error
	var decodeErr error

	// read JSON data from file
	// and try to decompress the data
	if dec, err := zstd.NewReader(f); err == nil {
		input, decodeErr = io.ReadAll(dec)
	}
	// if any error occured during decompress the data
	// just try to read the file directly
	if decodeErr != nil {
		f.Seek(0, io.SeekStart)
		input, readErr = io.ReadAll(f)
	}
	f.Close()
	if readErr != nil {
		log.Fatal(err)
	}

	// decode JSON
	var data promDump
	if err = json.Unmarshal(input, &data); err != nil {
		fmt.Println(string(input))
		log.Fatal(err)
	}

	if err := writeBatchPoints(data, opt); err != nil {
		log.Fatal(err)
	}
	return nil
}

// LoadMetrics reads the dumped metric JSON files and reload them
// to an influxdb instance.
func LoadMetrics(dataDir string, opt *RebuildOptions) error {
	// read cluster name
	clsName, err := os.ReadFile(path.Join(dataDir, fileNameClusterName))
	if err != nil {
		return err
	}
	opt.Cluster = string(clsName)

	// extract collection session id
	dirFields := strings.Split(dataDir, "-")
	opt.Session = dirFields[len(dirFields)-1]

	proms, err := os.ReadDir(path.Join(dataDir, subdirMonitor, subdirMetrics))
	if err != nil {
		return err
	}
	mtcFiles := make(map[string][]fs.DirEntry)
	for _, p := range proms {
		if !p.IsDir() {
			continue
		}
		subs, err := os.ReadDir(path.Join(dataDir, subdirMonitor, subdirMetrics, p.Name()))
		if err != nil {
			return err
		}
		mtcFiles[p.Name()] = subs
	}

	// load individual metric files
	mb := progress.NewMultiBar("Loading metrics")
	bars := make(map[string]*progress.MultiBarItem)
	for p := range mtcFiles {
		bars[p] = mb.AddBar(p)
	}
	mb.StartRenderLoop()
	defer mb.StopRenderLoop()

	// connect to influxdb
	client := newClient(opt)
	// create database has no side effect if database already exist
	_, err = queryDB(client, opt.DBName, fmt.Sprintf("CREATE DATABASE %s", opt.DBName))
	client.Close()
	if err != nil {
		log.Fatal(err)
	}

	var loadErr error
	tl := utils.NewTokenLimiter(uint(opt.Concurrency))
	for p, files := range mtcFiles {
		cnt := 0
		b := bars[p]
		go func(tok *utils.Token, parent string, files []fs.DirEntry) {
			total := len(files)
			for _, file := range files {
				if file.IsDir() {
					continue
				}
				cnt++
				b.UpdateDisplay(&progress.DisplayProps{
					Prefix: fmt.Sprintf(" - Loading metrics from %s", parent),
					Suffix: fmt.Sprintf("%d/%d: %s", cnt, total, file.Name()),
				})

				fOpt := *opt
				fOpt.File = path.Join(
					dataDir, subdirMonitor, subdirMetrics,
					parent, file.Name(),
				)
				if err := fOpt.LoadMetrics(); err != nil {
					b.UpdateDisplay(&progress.DisplayProps{
						Prefix: fmt.Sprintf(" - Load metrics from %s", parent),
						Suffix: err.Error(),
						Mode:   progress.ModeError,
					})
					if loadErr == nil {
						loadErr = err
						return
					}
				}
			}
			b.UpdateDisplay(&progress.DisplayProps{
				Prefix: fmt.Sprintf(" - Load metrics from %s", parent),
				Mode:   progress.ModeDone,
			})
			tl.Put(tok)
		}(tl.Get(), p, files)
	}
	tl.Wait()

	return loadErr
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
	opts *RebuildOptions,
) []*influx.Point {
	// build tags
	rawTags := series.Metric
	tags := make(map[string]string)
	for k, v := range rawTags {
		tags[string(k)] = string(v)
	}
	tags["cluster"] = opts.Cluster
	tags["session"] = opts.Session
	tags["monitor"] = "prometheus"
	measurement := tags["__name__"]

	// build points
	ptList := make([]*influx.Point, 0)
	for _, point := range series.Values {
		timestamp := point.Timestamp.Time()
		fields := map[string]interface{}{
			// model.SampleValue is alias of float64
			"value": float64(point.Value),
		}
		if pt, err := influx.NewPoint(measurement, tags, fields, timestamp); err == nil {
			ptList = append(ptList, pt)
		} // errored points are ignored
	}

	return ptList
}

func writeBatchPoints(data promDump, opts *RebuildOptions) error {
	tl := utils.NewTokenLimiter(uint(opts.Concurrency) + 1)
	errChan := make(chan error)
	for _, series := range data.Data.Result {
		ptList := buildPoints(series, opts)

		for _, chunk := range slicePoints(ptList, opts.Chunk) {
			go func(tok *utils.Token, chunk []*influx.Point) {
				defer tl.Put(tok)

				bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
					Database:  opts.DBName,
					Precision: "s",
				})
				if err != nil {
					errChan <- err
					return
				}
				bp.AddPoints(chunk)

				// create influx.Client and close it every time we write a BatchPoints
				// series to reduce memory usage on large dataset
				client := newClient(opts)
				defer client.Close()

				// write batch points to influxdb
				if err := client.Write(bp); err != nil {
					errChan <- err
					return
				}
			}(tl.Get(), chunk)
		}
	}

	select {
	case err := <-errChan:
		return err
	default:
		tl.Wait()
		return nil
	}
}
