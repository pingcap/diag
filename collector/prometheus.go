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
	"github.com/pingcap/tiup/pkg/cluster/spec"
	log "github.com/sirupsen/logrus"
)

const (
	subdirMonitor = "monitor"
	subdirAlerts  = "alerts"
	subdirMetrics = "metrics"
)

// collectAlerts gathers alert list from Prometheus
func collectAlerts(
	topo *spec.Specification,
	resultDir string,
) error {
	if len(topo.Monitors) < 1 {
		fmt.Println("No Prometheus node found in topology, skip.")
		return nil
	}

	var queryOK bool
	var queryErr error
	for _, prom := range topo.Monitors {
		promAddr := fmt.Sprintf("%s:%d", prom.Host, prom.Port)
		if err := ensureMonitorDir(resultDir, subdirAlerts, fmt.Sprintf("%s-%d", prom.Host, prom.Port)); err != nil {
			return err
		}

		c := &http.Client{Timeout: time.Second * 10}
		resp, err := c.PostForm(fmt.Sprintf("http://%s/api/v1/query", promAddr), url.Values{"query": {"ALERTS"}})
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		f, err := os.Create(filepath.Join(resultDir, subdirMonitor, subdirAlerts, fmt.Sprintf("%s-%d", prom.Host, prom.Port), "alerts.json"))
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

// collectMetrics gathers metrics from Prometheus
func collectMetrics(
	topo *spec.Specification,
	opt *BaseOptions,
	resultDir string,
) error {
	if len(topo.Monitors) < 1 {
		fmt.Println("No Prometheus node found in topology, skip.")
		return nil
	}

	currTime := time.Now()
	end := opt.ScrapeEnd
	if end == "" {
		end = currTime.Format(time.RFC3339)
	}
	begin := opt.ScrapeBegin
	if begin == "" {
		begin = currTime.Add(time.Duration(-1) * time.Hour).Format(time.RFC3339)
	}

	var queryOK bool
	var queryErr error
	tl := utils.NewTokenLimiter(uint(runtime.NumCPU()))
	for _, prom := range topo.Monitors {
		promAddr := fmt.Sprintf("%s:%d", prom.Host, prom.Port)
		if err := ensureMonitorDir(resultDir, subdirMetrics, fmt.Sprintf("%s-%d", prom.Host, prom.Port)); err != nil {
			return err
		}

		c := &http.Client{Timeout: time.Second * 10}
		metrics, err := getMetricList(c, promAddr)
		if err == nil {
			queryOK = queryOK || true
		}
		queryErr = err

		for _, mtc := range metrics {
			go func(tok *utils.Token, mtc string) {
				collectMetric(c, prom, begin, end, mtc, resultDir)
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
