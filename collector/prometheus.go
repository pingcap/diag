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
	maxQueryRange = 120 * 60 // 120min
	minQueryRange = 5 * 60   // 5min
)

type collectMonitor struct {
	Metric bool
	Alert  bool
}

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
	if m.mode != CollectModeManual && len(topo.Monitors) < 1 {
		fmt.Println("No monitoring node (prometheus) found in topology, skip.")
		return nil
	}

	monitors := make([]string, 0)
	if eps, found := topo.Attributes[AttrKeyPromEndpoint]; found {
		monitors = append(monitors, eps.([]string)...)
	} else {
		for _, prom := range topo.Monitors {
			monitors = append(monitors, fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort()))
		}
	}

	var queryOK bool
	var queryErr error
	for _, promAddr := range monitors {
		if err := ensureMonitorDir(c.resultDir, subdirAlerts, strings.ReplaceAll(promAddr, ":", "-")); err != nil {
			return err
		}

		client := &http.Client{Timeout: time.Second * 10}
		resp, err := client.PostForm(fmt.Sprintf("http://%s/api/v1/query", promAddr), url.Values{"query": {"ALERTS"}})
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		f, err := os.Create(filepath.Join(c.resultDir, subdirMonitor, subdirAlerts, strings.ReplaceAll(promAddr, ":", "-"), "alerts.json"))
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
	metrics   []string // metric list
	filter    []string
	limit     int // series*min per query
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
	if m.mode != CollectModeManual && len(topo.Monitors) < 1 {
		if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
			fmt.Println("No Prometheus node found in topology, skip.")
		} else {
			m.logger.Warnf("No Prometheus node found in topology, skip.")
		}
		return nil, nil
	}

	tsEnd, _ := utils.ParseTime(c.GetBaseOptions().ScrapeEnd)
	tsStart, _ := utils.ParseTime(c.GetBaseOptions().ScrapeBegin)
	nsec := tsEnd.Unix() - tsStart.Unix()

	monitors := make([]string, 0)
	if eps, found := topo.Attributes[AttrKeyPromEndpoint]; found {
		monitors = append(monitors, eps.([]string)...)
	} else {
		for _, prom := range topo.Monitors {
			monitors = append(monitors, fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort()))
		}
	}

	var queryErr error
	var promAddr string
	for _, prom := range monitors {
		promAddr = prom
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
	if m.mode != CollectModeManual && len(topo.Monitors) < 1 {
		if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
			fmt.Println("No Prometheus node found in topology, skip.")
		} else {
			m.logger.Warnf("No Prometheus node found in topology, skip.")
		}
		return nil
	}

	monitors := make([]string, 0)
	if eps, found := topo.Attributes[AttrKeyPromEndpoint]; found {
		monitors = append(monitors, eps.([]string)...)
	} else {
		for _, prom := range topo.Monitors {
			monitors = append(monitors, fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort()))
		}
	}

	mb := progress.NewMultiBar("+ Dumping metrics")
	bars := make(map[string]*progress.MultiBarItem)
	total := len(c.metrics)
	mu := sync.Mutex{}
	for _, prom := range monitors {
		key := prom
		if _, ok := bars[key]; !ok {
			bars[key] = mb.AddBar(fmt.Sprintf("  - Querying server %s", key))
		}
	}
	switch m.mode {
	case CollectModeTiUP,
		CollectModeManual:
		mb.StartRenderLoop()
		defer mb.StopRenderLoop()
	}

	qLimit := c.opt.Concurrency
	cpuCnt := runtime.NumCPU()
	if cpuCnt < qLimit {
		qLimit = cpuCnt
	}
	tl := utils.NewTokenLimiter(uint(qLimit))

	for _, prom := range monitors {
		key := prom
		done := 1

		if err := ensureMonitorDir(c.resultDir, subdirMetrics, strings.ReplaceAll(prom, ":", "-")); err != nil {
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

				tsEnd, _ := utils.ParseTime(c.GetBaseOptions().ScrapeEnd)
				tsStart, _ := utils.ParseTime(c.GetBaseOptions().ScrapeBegin)
				collectMetric(m.logger, client, prom, tsStart, tsEnd, mtc, c.resultDir, c.limit)

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

func getSeriesNum(c *http.Client, promAddr, metric string) (int, error) {
	resp, err := c.PostForm(
		fmt.Sprintf("http://%s/api/v1/series", promAddr),
		url.Values{
			"match[]": {metric},
		},
	)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode/100 != 2 {
		return 0, fmt.Errorf("fail to get series. Status Code %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	r := struct {
		Series []interface{} `json:"data"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, err
	}
	return len(r.Series), nil
}

func collectMetric(
	l *logprinter.Logger,
	c *http.Client,
	promAddr string,
	beginTime, endTime time.Time,
	mtc, resultDir string,
	speedlimit int,
) {
	l.Debugf("Querying series of %s...", mtc)
	series, err := getSeriesNum(c, promAddr, mtc)
	if err != nil {
		l.Errorf("%s", err)
		return
	}

	// split time into smaller ranges to avoid querying too many data in one request
	if speedlimit == 0 {
		speedlimit = 10000
	}
	block := 3600 * speedlimit / series
	if block > maxQueryRange {
		block = maxQueryRange
	}
	if block < minQueryRange {
		block = minQueryRange
	}

	l.Debugf("Dumping metric %s-%s-%s...", mtc, beginTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	for queryEnd := endTime; queryEnd.After(beginTime); queryEnd = queryEnd.Add(time.Duration(-block) * time.Second) {
		querySec := block
		queryBegin := queryEnd.Add(time.Duration(-block) * time.Second)
		if queryBegin.Before(beginTime) {
			querySec = int(queryEnd.Sub(beginTime).Seconds())
			queryBegin = beginTime
		}
		if err := tiuputils.Retry(
			func() error {
				resp, err := c.PostForm(
					fmt.Sprintf("http://%s/api/v1/query", promAddr),
					url.Values{
						"query": {fmt.Sprintf("%s[%ds]", mtc, querySec)},
						"time":  {queryEnd.Format(time.RFC3339)},
					},
				)
				if err != nil {
					l.Errorf("failed query metric %s: %s, retry...", mtc, err)
					return err
				}
				// Prometheus API response format is JSON. Every successful API request returns a 2xx status code.
				if resp.StatusCode/100 != 2 {
					l.Errorf("failed query metric %s: Status Code %d, retry...", mtc, resp.StatusCode)
				}
				defer resp.Body.Close()

				dst, err := os.Create(
					filepath.Join(
						resultDir, subdirMonitor, subdirMetrics, strings.ReplaceAll(promAddr, ":", "-"),
						fmt.Sprintf("%s_%s_%s.json", mtc, queryBegin.Format(time.RFC3339), queryEnd.Format(time.RFC3339)),
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
				l.Debugf(" Dumped metric %s from %s to %s (%d bytes)", mtc, queryBegin.Format(time.RFC3339), queryEnd.Format(time.RFC3339), n)
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
