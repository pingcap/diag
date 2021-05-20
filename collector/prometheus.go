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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/pingcap/tidb-foresight/utils"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	log "github.com/sirupsen/logrus"
)

const (
	subdirMonitor = "monitor"
	subdirAlerts  = "alerts"
	subdirMetrics = "metrics"
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

// Collect implements the Collector interface
func (c *AlertCollectOptions) Collect(topo *spec.Specification) error {
	if len(topo.Monitors) < 1 {
		fmt.Println("No Prometheus node found in topology, skip.")
		return nil
	}

	var queryOK bool
	var queryErr error
	for _, prom := range topo.Monitors {
		promAddr := fmt.Sprintf("%s:%d", prom.Host, prom.Port)
		if err := ensureMonitorDir(c.resultDir, subdirAlerts, fmt.Sprintf("%s-%d", prom.Host, prom.Port)); err != nil {
			return err
		}

		client := &http.Client{Timeout: time.Second * 10}
		resp, err := client.PostForm(fmt.Sprintf("http://%s/api/v1/query", promAddr), url.Values{"query": {"ALERTS"}})
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		f, err := os.Create(filepath.Join(c.resultDir, subdirMonitor, subdirAlerts, fmt.Sprintf("%s-%d", prom.Host, prom.Port), "alerts.json"))
		if err == nil {
			queryOK = queryOK || true
		} else {
			queryErr = err
		}
		defer f.Close()

		if _, err := io.Copy(f, resp.Body); err != nil {
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

// Collect implements the Collector interface
func (c *MetricCollectOptions) Collect(topo *spec.Specification) error {
	if len(topo.Monitors) < 1 {
		fmt.Println("No Prometheus node found in topology, skip.")
		return nil
	}

	begin, end, err := parseTimeRange(c.GetBaseOptions().ScrapeBegin, c.GetBaseOptions().ScrapeEnd)
	if err != nil {
		return err
	}

	var queryOK bool
	var queryErr error
	tl := utils.NewTokenLimiter(uint(runtime.NumCPU()))
	for _, prom := range topo.Monitors {
		promAddr := fmt.Sprintf("%s:%d", prom.Host, prom.Port)
		if err := ensureMonitorDir(c.resultDir, subdirMetrics, fmt.Sprintf("%s-%d", prom.Host, prom.Port)); err != nil {
			return err
		}

		client := &http.Client{Timeout: time.Second * 10}
		metrics, err := getMetricList(client, promAddr)
		if err == nil {
			queryOK = queryOK || true
		}
		queryErr = err

		for _, mtc := range metrics {
			go func(tok *utils.Token, mtc string) {
				collectMetric(client, prom, begin, end, mtc, c.resultDir)
				tl.Put(tok)
			}(tl.Get(), mtc)
		}
	}
	tl.Wait()

	// if query successed for any one of prometheus, ignore errors for other instances
	if !queryOK {
		return queryErr
	}
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

func collectMetric(c *http.Client, prom *spec.PrometheusSpec, begin, end, mtc, resultDir string) {
	promAddr := fmt.Sprintf("%s:%d", prom.Host, prom.Port)
	step := 15 // use 15s step

	resp, err := c.PostForm(
		fmt.Sprintf("http://%s/api/v1/query_range", promAddr),
		url.Values{
			"query": {mtc},
			"start": {begin},
			"end":   {end},
			"step":  {strconv.Itoa(step)},
		},
	)
	if err != nil {
		log.Errorf("collect metric %s: %w", mtc, err)
		return
	}
	defer resp.Body.Close()

	dst, err := os.Create(
		filepath.Join(
			resultDir, subdirMonitor, subdirMetrics, fmt.Sprintf("%s-%d", prom.Host, prom.Port),
			fmt.Sprintf("%s_%s_%s.json", mtc, begin, end),
		),
	)
	if err != nil {
		log.Errorf("collect metric %s: %w", mtc, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, resp.Body); err != nil {
		log.Errorf("collect metric %s: %w", mtc, err)
	}
}

func ensureMonitorDir(base string, sub ...string) error {
	e := []string{base, subdirMonitor}
	e = append(e, sub...)
	dir := path.Join(e...)
	return os.MkdirAll(dir, 0755)
}

func parseTimeRange(scrapeStart, scrapeEnd string) (string, string, error) {
	currTime := time.Now()

	end := scrapeEnd
	if end == "" {
		end = currTime.Format(time.RFC3339)
	}
	tsEnd, err := utils.ParseTime(end)
	if err != nil {
		return "", "", err
	}
	end = tsEnd.Format(time.RFC3339)

	begin := scrapeStart
	if begin == "" {
		begin = tsEnd.Add(time.Duration(-1) * time.Hour).Format(time.RFC3339)
	}
	tsStart, err := utils.ParseTime(begin)
	if err != nil {
		return "", "", err
	}
	begin = tsStart.Format(time.RFC3339)

	if !tsStart.Before(tsEnd) {
		return "", "", fmt.Errorf("start time must be earlier than end time")
	}
	return begin, end, nil
}
