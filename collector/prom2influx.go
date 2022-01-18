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
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"

	influx "github.com/influxdata/influxdb/client/v2"
	json "github.com/json-iterator/go"
	"github.com/klauspost/compress/zstd"
	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/tiup/pkg/tui/progress"
	"github.com/prometheus/common/model"
)

// LoadMetrics reads the dumped metric JSON files and reload them
// to an influxdb instance.
func LoadMetrics(ctx context.Context, dataDir string, opt *RebuildOptions) error {
	// read cluster name
	body, err := os.ReadFile(path.Join(dataDir, FileNameClusterJSON))
	if err != nil {
		return err
	}
	clusterJSON := map[string]interface{}{}
	err = json.Unmarshal(body, &clusterJSON)
	if err != nil {
		return err
	}
	opt.Cluster = clusterJSON["cluster_name"].(string)

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
	client, err := newClient(opt)
	if err != nil {
		return err
	}
	// create database has no side effect if database already exist
	_, err = queryDB(client, opt.DBName, fmt.Sprintf("CREATE DATABASE %s", opt.DBName))
	client.Close()
	if err != nil {
		return err
	}

	errChan := make(chan error)
	doneChan := make(chan struct{}, 1)
	tl := utils.NewTokenLimiter(uint(opt.Concurrency) + 1)
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
				if err := fOpt.LoadMetrics(tl); err != nil {
					b.UpdateDisplay(&progress.DisplayProps{
						Prefix: fmt.Sprintf(" - Load metrics from %s", parent),
						Suffix: err.Error(),
						Mode:   progress.ModeError,
					})
					errChan <- err
					return
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
	doneChan <- struct{}{}

	select {
	case <-ctx.Done():
		return nil
	case err := <-errChan:
		return err
	case <-doneChan:
		return nil
	}
}

func (opt *RebuildOptions) LoadMetrics(tl *utils.TokenLimiter) error {
	f, err := os.Open(opt.File)
	if err != nil {
		return err
	}
	defer f.Close()

	var input []byte
	var readErr error
	var decodeErr error

	// read JSON data from file
	// and try to decompress the data
	if dec, err := zstd.NewReader(f); err == nil {
		defer dec.Close()
		input, decodeErr = io.ReadAll(dec)
	}
	// if any error occurred during decompressing the data
	// just try to read the file directly
	if decodeErr != nil {
		f.Seek(0, io.SeekStart)
		input, readErr = io.ReadAll(f)
	}
	if readErr != nil {
		return err
	}

	// decode JSON
	var data promDump
	if err = json.Unmarshal(input, &data); err != nil {
		//fmt.Println(string(input))
		return err
	}

	return writeBatchPoints(tl, data, opt)

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
func queryDB(clnt influx.Client, dbName string, cmd string) (res []influx.Result, err error) {
	q := influx.Query{
		Command:  cmd,
		Database: dbName,
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

func slicePoints(data chan *influx.Point, chunkSize int) chan []*influx.Point {
	result := make(chan []*influx.Point)

	go func() {
		i := 0
		chunk := make([]*influx.Point, 0)
		for p := range data {
			chunk = append(chunk, p)
			if i >= chunkSize { // truncate
				result <- chunk
				chunk = make([]*influx.Point, 0)
				i = 0
			}
			i++
		}
		if len(chunk) > 0 { // tailing points
			result <- chunk
		}
		close(result)
	}()

	return result
}

func newClient(opts *RebuildOptions) (influx.Client, error) {
	// connect to influxdb
	client, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     fmt.Sprintf("http://%s:%d", opts.Host, opts.Port),
		Username: opts.User,
		Password: opts.Passwd,
	})
	return client, err
}

func buildPoints(
	tl *utils.TokenLimiter,
	series *model.SampleStream,
	opts *RebuildOptions,
) chan *influx.Point {
	// build tags
	tags := make(map[string]string)
	for k, v := range series.Metric {
		tags[string(k)] = string(v)
	}
	tags["cluster"] = opts.Cluster
	tags["session"] = opts.Session
	tags["monitor"] = "prometheus"
	measurement := tags["__name__"]

	// build points
	ptChan := make(chan *influx.Point)
	go func() {
		wg := sync.WaitGroup{}
		for _, point := range series.Values {
			wg.Add(1)
			go func(tok *utils.Token, point model.SamplePair) {
				timestamp := point.Timestamp.Time()
				fields := map[string]interface{}{
					// model.SampleValue is alias of float64
					"value": float64(point.Value),
				}
				if pt, err := influx.NewPoint(measurement, tags, fields, timestamp); err == nil {
					ptChan <- pt
				} // errored points are ignored
				tl.Put(tok)
				wg.Done()
			}(tl.Get(), point)
		}
		wg.Wait()
		close(ptChan)
	}()

	return ptChan
}

func writeBatchPoints(tl *utils.TokenLimiter, data promDump, opts *RebuildOptions) error {
	// build and write points
	errChan := make(chan error)
	doneChan := make(chan struct{}, 1)
	wg := sync.WaitGroup{}
	for _, series := range data.Data.Result {
		ptChan := buildPoints(tl, series, opts)

		for chunk := range slicePoints(ptChan, opts.Chunk) {
			wg.Add(1)
			go func(tok *utils.Token, chunk []*influx.Point) {
				defer tl.Put(tok)
				defer wg.Done()

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
				client, err := newClient(opts)
				if err != nil {
					client.Close()
					errChan <- err
					return
				}
				defer client.Close()

				// write batch points to influxdb
				if err := client.Write(bp); err != nil {
					errChan <- err
					return
				}
			}(tl.Get(), chunk)
		}
	}
	wg.Wait()
	doneChan <- struct{}{}

	select {
	case err := <-errChan:
		return err
	case <-doneChan:
		return nil
	}
}
