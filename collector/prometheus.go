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

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	"github.com/klauspost/compress/zstd"
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/diag/pkg/utils"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/tui/progress"
	tiuputils "github.com/pingcap/tiup/pkg/utils"
)

const (
	subdirMonitor = "monitor"
	subdirAlerts  = "alerts"
	subdirMetrics = "metrics"
	metricStep    = 15 // use 15s step
)

// AlertCollectOptions is the options collecting alerts
type AlertCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	resultDir string
}

// Desc implements the Collector interface
func (c *AlertCollectOptions) Desc() string {
	return "alert lists from Prometheus node"
}

// GetBaseOptions implements the Collector interface
func (c *AlertCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *AlertCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *AlertCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *AlertCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *AlertCollectOptions) Prepare(_ *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *AlertCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	if len(topo.Monitors) < 1 {
		fmt.Println("No monitoring node (prometheus) found in topology, skip.")
		return nil
	}

	var queryOK bool
	var queryErr error
	for _, prom := range topo.Monitors {
		promAddr := fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort())
		if err := ensureMonitorDir(c.resultDir, subdirAlerts, fmt.Sprintf("%s-%d", prom.Host(), prom.MainPort())); err != nil {
			return err
		}

		client := &http.Client{Timeout: time.Second * 10}
		resp, err := client.PostForm(fmt.Sprintf("http://%s/api/v1/query", promAddr), url.Values{"query": {"ALERTS"}})
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		f, err := os.Create(filepath.Join(c.resultDir, subdirMonitor, subdirAlerts, fmt.Sprintf("%s-%d", prom.Host(), prom.MainPort()), "alerts.json"))
		if err == nil {
			queryOK = queryOK || true
		} else {
			queryErr = err
		}
		defer f.Close()

		enc, err := zstd.NewWriter(f)
		if err != nil {
			m.logger.Errorf("failed compressing alert list: %s, retry...\n", err)
			return err
		}
		defer enc.Close()
		_, err = io.Copy(enc, resp.Body)
		if err != nil {
			m.logger.Errorf("failed writing alert list to file: %s, retry...\n", err)
			return err
		}
	}

	// if query successed for any one of prometheus, ignore errors for other instances
	if !queryOK {
		return queryErr
	}
	return nil
}

// MetricCollectOptions is the options collecting metrics
type MetricCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	resultDir string
	timeSteps []string
	metrics   []string // metric list
	filter    []string
}

// Desc implements the Collector interface
func (c *MetricCollectOptions) Desc() string {
	return "metrics from Prometheus node"
}

// GetBaseOptions implements the Collector interface
func (c *MetricCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *MetricCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *MetricCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *MetricCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *MetricCollectOptions) Prepare(m *Manager, topo *models.TiDBCluster) (map[string][]CollectStat, error) {
	if len(topo.Monitors) < 1 {
		if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
			fmt.Println("No Prometheus node found in topology, skip.")
		} else {
			m.logger.Warnf("No Prometheus node found in topology, skip.")
		}
		return nil, nil
	}

	ts, nsec, err := parseTimeRange(c.GetBaseOptions().ScrapeBegin, c.GetBaseOptions().ScrapeEnd)
	if err != nil {
		return nil, err
	}
	c.timeSteps = ts

	var queryErr error
	var promAddr string
	for _, prom := range topo.Monitors {
		promAddr = fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort())
		client := &http.Client{Timeout: time.Second * 10}
		c.metrics, queryErr = getMetricList(client, promAddr)
		if queryErr == nil {
			break
		}
	}
	// if query successed for any one of prometheus, ignore errors for other instances
	if queryErr != nil {
		return nil, queryErr
	}

	c.metrics = filterMetrics(c.metrics, c.filter)

	result := make(map[string][]CollectStat)
	insCnt := len(topo.Components())
	cStat := CollectStat{
		Target: fmt.Sprintf("%d metrics, compressed", len(c.metrics)),
		Size:   int64(11*len(c.metrics)*insCnt) * nsec, // empirical formula, inaccurate
	}
	// compression rate is approximately 2.5%
	cStat.Size = int64(float64(cStat.Size) * 0.025)

	result[promAddr] = append(result[promAddr], cStat)

	return result, nil
}

// Collect implements the Collector interface
func (c *MetricCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	if len(topo.Monitors) < 1 {
		if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
			fmt.Println("No Prometheus node found in topology, skip.")
		} else {
			m.logger.Warnf("No Prometheus node found in topology, skip.")
		}
		return nil
	}

	mb := progress.NewMultiBar("+ Dumping metrics")
	bars := make(map[string]*progress.MultiBarItem)
	total := len(c.metrics)
	mu := sync.Mutex{}
	for _, prom := range topo.Monitors {
		key := fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort())
		if _, ok := bars[key]; !ok {
			bars[key] = mb.AddBar(fmt.Sprintf("  - Querying server %s", key))
		}
	}
	if m.mode == CollectModeTiUP {
		mb.StartRenderLoop()
		defer mb.StopRenderLoop()
	}

	qLimit := c.opt.Concurrency
	cpuCnt := runtime.NumCPU()
	if cpuCnt < qLimit {
		qLimit = cpuCnt
	}
	tl := utils.NewTokenLimiter(uint(qLimit))
	for _, prom := range topo.Monitors {
		key := fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort())
		done := 1

		if err := ensureMonitorDir(c.resultDir, subdirMetrics, fmt.Sprintf("%s-%d", prom.Host(), prom.MainPort())); err != nil {
			bars[key].UpdateDisplay(&progress.DisplayProps{
				Prefix: fmt.Sprintf("  - Query server %s: %s", key, err),
				Mode:   progress.ModeError,
			})
			return err
		}

		client := &http.Client{Timeout: time.Second * 60}

		for _, mtc := range c.metrics {
			go func(tok *utils.Token, mtc string) {
				bars[key].UpdateDisplay(&progress.DisplayProps{
					Prefix: fmt.Sprintf("  - Querying server %s", key),
					Suffix: fmt.Sprintf("%d/%d querying %s ...", done, total, mtc),
				})

				collectMetric(m.logger, client, prom, c.timeSteps, mtc, c.resultDir)

				mu.Lock()
				done++
				if done >= total {
					bars[key].UpdateDisplay(&progress.DisplayProps{
						Prefix: fmt.Sprintf("  - Query server %s", key),
						Mode:   progress.ModeDone,
					})
				}
				mu.Unlock()

				tl.Put(tok)
			}(tl.Get(), mtc)
		}
	}

	tl.Wait()

	return nil
}

func getMetricList(c *http.Client, prom string) ([]string, error) {
	resp, err := c.Get(fmt.Sprintf("http://%s/api/v1/label/__name__/values", prom))
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	r := struct {
		Metrics []string `json:"data"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return []string{}, err
	}
	return r.Metrics, nil
}

func collectMetric(
	l *logprinter.Logger,
	c *http.Client,
	prom *models.MonitorSpec,
	ts []string,
	mtc, resultDir string,
) {
	l.Debugf("Dumping metric %s...", mtc)

	promAddr := fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort())

	for i := 0; i < len(ts)-1; i++ {
		if err := tiuputils.Retry(
			func() error {
				resp, err := c.PostForm(
					fmt.Sprintf("http://%s/api/v1/query_range", promAddr),
					url.Values{
						"query": {mtc},
						"start": {ts[i]},
						"end":   {ts[i+1]},
						"step":  {strconv.Itoa(metricStep)},
					},
				)
				if err != nil {
					l.Errorf("failed query metric %s: %s, retry...", mtc, err)
					return err
				}
				defer resp.Body.Close()

				dst, err := os.Create(
					filepath.Join(
						resultDir, subdirMonitor, subdirMetrics, fmt.Sprintf("%s-%d", prom.Host(), prom.MainPort()),
						fmt.Sprintf("%s_%s_%s.json", mtc, ts[i], ts[i+1]),
					),
				)
				if err != nil {
					l.Errorf("collect metric %s: %s, retry...", mtc, err)
				}
				defer dst.Close()

				// compress the metric
				var n int64
				enc, err := zstd.NewWriter(dst)
				if err != nil {
					l.Errorf("failed compressing metric %s: %s, retry...\n", mtc, err)
					return err
				}
				defer enc.Close()
				n, err = io.Copy(enc, resp.Body)
				if err != nil {
					l.Errorf("failed writing metric %s to file: %s, retry...\n", mtc, err)
					return err
				}
				l.Debugf(" Dumped metric %s from %s to %s (%d bytes)", mtc, ts[i], ts[i+1], n)
				return nil
			},
			tiuputils.RetryOption{
				Attempts: 3,
				Delay:    time.Microsecond * 300,
				Timeout:  time.Second * 120,
			},
		); err != nil {
			l.Errorf("Error quering metrics %s: %s", mtc, err)
		}
	}
}

func ensureMonitorDir(base string, sub ...string) error {
	e := []string{base, subdirMonitor}
	e = append(e, sub...)
	dir := path.Join(e...)
	return os.MkdirAll(dir, 0755)
}

func parseTimeRange(scrapeStart, scrapeEnd string) ([]string, int64, error) {
	currTime := time.Now()

	end := scrapeEnd
	if end == "" {
		end = currTime.Format(time.RFC3339)
	}
	tsEnd, err := utils.ParseTime(end)
	if err != nil {
		return nil, 0, err
	}

	begin := scrapeStart
	if begin == "" {
		begin = tsEnd.Add(time.Duration(-1) * time.Hour).Format(time.RFC3339)
	}
	tsStart, err := utils.ParseTime(begin)
	if err != nil {
		return nil, 0, err
	}

	// split time into smaller ranges to avoid querying too many data
	// in one request
	ts := []string{tsStart.Format(time.RFC3339)}
	block := time.Second * 3600 * 2
	cursor := tsStart
	for {
		if cursor.After(tsEnd) {
			ts = append(ts, tsEnd.Format(time.RFC3339))
			break
		}
		next := cursor.Add(block)
		if next.Before(tsEnd) {
			ts = append(ts, cursor.Format(time.RFC3339))
		} else {
			ts = append(ts, tsEnd.Format(time.RFC3339))
			break
		}
		cursor = next
	}

	return ts, tsEnd.Unix() - tsStart.Unix(), nil
}

func filterMetrics(src, filter []string) []string {
	if filter == nil {
		return src
	}
	var res []string
	for _, metric := range src {
		for _, prefix := range filter {
			if strings.HasPrefix(metric, prefix) {
				res = append(res, metric)
			}
		}
	}
	return res
}
