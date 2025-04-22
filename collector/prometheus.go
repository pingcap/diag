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
	"context"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/joomcode/errorx"
	json "github.com/json-iterator/go"
	"github.com/klauspost/compress/zstd"
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/diag/pkg/utils"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/tui/progress"
	tiuputils "github.com/pingcap/tiup/pkg/utils"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	subdirMonitor   = "monitor"
	subdirAlerts    = "alerts"
	subdirMetrics   = "metrics"
	subdirRaw       = "raw"
	maxQueryRange   = 120 * 60 // 120min
	smallQueryRange = 15       // 15s
	logQuerySeries  = 120000   // The value is equal to the result of 3600*speedLimit/300(s), where the default value of speedLimit is 10000.
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
	compress  bool
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
func (c *AlertCollectOptions) Prepare(m *Manager, topo *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *AlertCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	if c.Kubeconfig != "" && m.diagMode == DiagModeCmd {
		// ignore collect alerts for "diag collectk"
		return nil
	}
	if m.mode != CollectModeManual && len(topo.Monitors) < 1 {
		m.logger.Warnf("No monitoring node (prometheus) found in topology, skip collecting alert.")
		return nil
	}

	monitors := make([]string, 0)
	if eps, found := topo.Attributes[AttrKeyPromEndpoint]; found && len(eps.([]string)) > 0 && eps.([]string)[0] != "" {
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

		client := &http.Client{Timeout: time.Second * time.Duration(c.opt.APITimeout)}
		resp, err := client.PostForm(fmt.Sprintf("http://%s/api/v1/query", promAddr), url.Values{"query": {"ALERTS"}})
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		f, err := os.Create(filepath.Join(c.resultDir, subdirMonitor, subdirAlerts, strings.ReplaceAll(promAddr, ":", "-"), "alerts.json"))
		if err == nil {
			queryOK = true
		} else {
			queryErr = err
		}
		defer f.Close()

		var enc io.WriteCloser
		if c.compress {
			enc, err = zstd.NewWriter(f)
			if err != nil {
				m.logger.Errorf("failed compressing alert list: %s, retry...\n", err)
				return err
			}
			defer enc.Close()
		} else {
			enc = f
		}
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
	opt          *operator.Options // global operations from cli
	resultDir    string
	label        map[string]string
	metrics      []string // metric list
	filter       []string
	exclude      []string
	lowPriority  []string
	limit        int // series*min per query
	minInterval  int // the minimum interval of a single request in seconds
	compress     bool
	customHeader []string
	endpoint     string
	portForward  bool
	stopChans    []chan struct{}
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

// Close implements the Collector interface
func (c *MetricCollectOptions) Close() {
	for _, c := range c.stopChans {
		if c != nil {
			close(c)
		}
	}
}

// Prepare implements the Collector interface
func (c *MetricCollectOptions) Prepare(m *Manager, topo *models.TiDBCluster) (map[string][]CollectStat, error) { // only collect from the first one
	if eps, found := topo.Attributes[AttrKeyPromEndpoint]; found && len(eps.([]string)) > 0 && eps.([]string)[0] != "" {
		c.endpoint = eps.([]string)[0]
	} else if len(topo.Monitors) > 0 {
		prom := topo.Monitors[0]
		if c.portForward {
			podName, _ := prom.Attributes()["pod"].(string)
			stopChan, port, err := c.NewForwardPorts(podName, 9090)
			if err != nil {
				return nil, err
			}
			c.stopChans = append(c.stopChans, stopChan)
			c.endpoint = fmt.Sprintf("127.0.0.1:%d", port)
		} else {
			c.endpoint = fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort())
		}
	} else {
		m.logger.Warnf("No Prometheus node found in topology, skip.")
		return nil, nil
	}

	tsEnd, _ := utils.ParseTime(c.GetBaseOptions().ScrapeEnd)
	tsStart, _ := utils.ParseTime(c.GetBaseOptions().ScrapeBegin)
	nsec := tsEnd.Unix() - tsStart.Unix()

	client := &http.Client{Timeout: time.Second * time.Duration(c.opt.APITimeout)}
	if err := tiuputils.Retry(
		func() error {
			var queryErr error
			c.metrics, queryErr = getMetricList(client, c.endpoint, c.customHeader)
			return queryErr
		},
		tiuputils.RetryOption{
			Attempts: 3,
			Delay:    time.Microsecond * 300,
			Timeout:  client.Timeout*3 + 5*time.Second, //make sure the retry timeout is longer than the api timeout
		},
	); err != nil {
		return nil, fmt.Errorf("failed to get metric list from %s: %s", c.endpoint, err)
	}

	c.metrics = filterMetrics(c.metrics, c.filter, c.exclude)

	result := make(map[string][]CollectStat)
	insCnt := len(topo.Components())
	cStat := CollectStat{
		Target: fmt.Sprintf("%d metrics, compressed", len(c.metrics)),
		Size:   int64(11*len(c.metrics)*insCnt) * nsec, // empirical formula, inaccurate
	}
	// compression rate is approximately 2.5%
	cStat.Size = int64(float64(cStat.Size) * 0.025)

	result[c.endpoint] = append(result[c.endpoint], cStat)

	return result, nil
}

// Collect implements the Collector interface
func (c *MetricCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	if c.endpoint == "" {
		return nil
	}
	startTime := time.Now()
	mb := progress.NewMultiBar("+ Dumping metrics")
	bars := make(map[string]*progress.MultiBarItem)

	key := c.endpoint
	if _, ok := bars[key]; !ok {
		bars[key] = mb.AddBar(fmt.Sprintf("  - Querying server %s", key))
	}

	if m.diagMode == DiagModeCmd {
		mb.StartRenderLoop()
		defer mb.StopRenderLoop()
	}

	qLimit := c.opt.Concurrency
	cpuCnt := runtime.NumCPU()
	if cpuCnt < qLimit {
		qLimit = cpuCnt
	}
	// Prometheus default query.max-concurrency is 20, so here set the max qLimit to 20.
	defaultQueryMaxConcurrency := 20
	if qLimit > defaultQueryMaxConcurrency {
		qLimit = defaultQueryMaxConcurrency
	}
	tl := utils.NewTokenLimiter(qLimit)

	if err := ensureMonitorDir(c.resultDir, subdirMetrics, strings.ReplaceAll(c.endpoint, ":", "-")); err != nil {
		bars[key].UpdateDisplay(&progress.DisplayProps{
			Prefix: fmt.Sprintf("  - Query server %s: %s", key, err),
			Mode:   progress.ModeError,
		})
		return err
	}

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:          defaultQueryMaxConcurrency,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
		Timeout: time.Second * time.Duration(c.opt.APITimeout),
	}

	if len(c.lowPriority) == 0 {
		c.collectMetrics(m.logger, client, c.metrics, midPriority, tl, bars)
		m.logger.Infof("Dumping metrics finish .......................................... token limit:%d, take time:%v",
			qLimit, time.Since(startTime))
	} else {
		c.collectMetrics(m.logger, client, c.metrics, highPriority, tl, bars)
		m.logger.Infof("Dumping high priority metrics finish .......................................... token limit:%d, concurrency:%d, take time:%v",
			qLimit, c.opt.Concurrency, time.Since(startTime))
		startTime = time.Now()
		c.collectMetrics(m.logger, client, c.lowPriority, lowPriority, tl, bars)
		m.logger.Infof("Dumping low priority metrics finish .......................................... token limit:%d, take time:%v",
			qLimit, time.Since(startTime))
	}

	return nil
}

const (
	lowPriority  = -1
	midPriority  = 0
	highPriority = 1
)

func (c *MetricCollectOptions) collectMetrics(
	l *logprinter.Logger,
	client *http.Client,
	metrics []string,
	priority int,
	tl *utils.TokenLimiter,
	bars map[string]*progress.MultiBarItem,
) {
	done := 1
	key := c.endpoint
	mu := sync.Mutex{}
	minInterval := c.minInterval
	if minInterval < smallQueryRange {
		minInterval = smallQueryRange
	}
	concurrency := 1
	total := len(c.metrics)
	if priority == highPriority {
		total = len(c.metrics) - len(c.lowPriority)
	} else if priority == lowPriority {
		total = len(c.lowPriority)
		concurrency = tl.Cap()
	}
	tsEnd, _ := utils.ParseTime(c.GetBaseOptions().ScrapeEnd)
	tsStart, _ := utils.ParseTime(c.GetBaseOptions().ScrapeBegin)
	originInfo := queryRangeInfo{
		queryBegin:  tsStart,
		queryEnd:    tsEnd,
		intervalSec: minInterval,
	}
	for _, mtc := range metrics {
		if priority == highPriority && utils.MatchPrefixs(mtc, c.lowPriority) {
			continue
		}

		go func(tok *utils.Token, mtc string) {
			bars[key].UpdateDisplay(&progress.DisplayProps{
				Prefix: fmt.Sprintf("  - Querying server %s", key),
				Suffix: fmt.Sprintf("%d/%d querying %s ...", done, total, mtc),
			})

			collectSingleMetric(l, client, key, originInfo, concurrency, mtc, c.label, c.resultDir, c.limit, c.compress, c.customHeader, "", tok.ID, tl)

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
	tl.Wait()
}

func getMetricList(c *http.Client, addr string, customHeader []string) ([]string, error) {
	return getAPIData[[]string](c, makeURL(addr, "/api/v1/label/__name__/values", nil), customHeader)
}

func getInstanceList(c *http.Client, addr string, queries map[string]string, customHeader []string) ([]string, error) {
	return getAPIData[[]string](c, makeURL(addr, "/api/v1/label/instance/values", queries), customHeader)
}

func getSeriesNum(c *http.Client, addr string, queries map[string]string, customHeader []string) (int, error) {
	series, err := getAPIData[[]interface{}](c, makeURL(addr, "/api/v1/series", queries), customHeader)
	if err != nil {
		return 0, err
	}
	return len(series), nil
}

func getAPIData[T any](c *http.Client, url string, customHeader []string) (T, error) {
	var body struct {
		Data T `json:"data"`
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return body.Data, err
	}
	utils.AddHeaders(req.Header, customHeader)
	resp, err := c.Do(req)
	if err != nil {
		return body.Data, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			return body.Data, fmt.Errorf("[%d] failed read body: %v", resp.StatusCode, err)
		}
		return body.Data, fmt.Errorf("[%d] %s", resp.StatusCode, string(msg))
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return body.Data, err
	}
	return body.Data, nil
}

func makeURL(addr string, path string, queries map[string]string) string {
	link := "http://" + addr + path
	if len(queries) == 0 {
		return link
	}
	vals := make(url.Values, len(queries))
	for k, v := range queries {
		vals[k] = []string{v}
	}
	return link + "?" + vals.Encode()
}

func collectSingleMetric(
	l *logprinter.Logger,
	c *http.Client,
	promAddr string,
	originInfo queryRangeInfo,
	concurrency int,
	mtc string,
	label map[string]string,
	resultDir string,
	speedLimit int,
	compress bool,
	customHeader []string,
	instance string,
	curTokenID int,
	tl *utils.TokenLimiter,
) {
	nameSuffix := ""
	if len(instance) > 0 {
		nameSuffix = "." + strings.ReplaceAll(instance, ":", "-")
	}
	query := generateQueryWitLabel(mtc, label)
	beginTime, endTime := originInfo.queryBegin, originInfo.queryEnd
	queries := map[string]string{
		"match[]": query,
		"start":   beginTime.Format(time.RFC3339),
		"end":     endTime.Format(time.RFC3339),
	}
	l.Debugf("Querying series of %s...", mtc+nameSuffix)

	var series int
	if err := tiuputils.Retry(
		func() error {
			// if len(instance) == 0 && rand.Float64() < 0.9 {
			// 	return errors.New("mock error")
			// }
			seriesNum, err := getSeriesNum(c, promAddr, queries, customHeader)
			series = seriesNum
			return err
		},
		tiuputils.RetryOption{
			Attempts: 3,
			Delay:    time.Microsecond * 300,
			Timeout:  c.Timeout*3 + 5*time.Second, //make sure the retry timeout is longer than the api timeout
		},
	); err != nil {
		l.Errorf("Failed to get series of %s: %s", mtc+nameSuffix, err)
		if len(instance) == 0 {
			// try by-instance dumping
			var instances []string
			if err := tiuputils.Retry(
				func() error {
					instanceLst, err := getInstanceList(c, promAddr, queries, customHeader)
					instances = instanceLst
					return err
				},
				tiuputils.RetryOption{
					Attempts: 3,
					Delay:    time.Microsecond * 300,
					Timeout:  c.Timeout*3 + 5*time.Second, //make sure the retry timeout is longer than the api timeout
				},
			); err != nil {
				l.Errorf("Failed to get instances of %s: %s", mtc, err)
				return
			}
			if len(instances) == 0 {
				l.Warnf("No instance found for %s", mtc)
				return
			}
			for _, instance := range instances {
				newLabel := make(map[string]string)
				maps.Copy(newLabel, label)
				newLabel["instance"] = instance
				collectSingleMetric(l, c, promAddr, originInfo, concurrency, mtc, newLabel, resultDir, speedLimit, compress, customHeader, instance, curTokenID, tl)
			}
		}
		return
	}

	if series <= 0 {
		l.Debugf("metric %s has %d series, ignore", mtc+nameSuffix, series)
		return
	}

	// split time into smaller ranges to avoid querying too many data in one request
	if speedLimit == 0 {
		speedLimit = 10000
	}
	block := 3600 * speedLimit / series
	if block > maxQueryRange {
		block = maxQueryRange
	}
	if block < originInfo.intervalSec {
		block = originInfo.intervalSec
	}

	if block == originInfo.intervalSec || series >= logQuerySeries {
		l.Infof("Collecting single metric %s series %d too large or interval %ds too small, so update concurrency from %d to %d, speedLimit:%d, req timeout:%v ...",
			mtc+nameSuffix, series, block, concurrency, tl.Cap(), speedLimit, c.Timeout)
		concurrency = tl.Cap()
	}
	retryOption := tiuputils.RetryOption{
		Attempts: 3,
		Delay:    time.Microsecond * 300,
		Timeout:  c.Timeout*3 + 5*time.Second, //make sure the retry timeout is longer than the api timeout
	}
	goCnt := 0
	qInfo := queryInfo{
		query:        query,
		promAddr:     promAddr,
		customHeader: customHeader,
		compress:     compress,
		retryOption:  retryOption,
	}
	queryInfoCh := make(chan queryInfo, concurrency)
	wg := WaitGroupWrapper{}
	startTime := time.Now()
	for queryEnd := endTime; queryEnd.After(beginTime); queryEnd = queryEnd.Add(time.Duration(-block) * time.Second) {
		querySec := block
		queryBegin := queryEnd.Add(time.Duration(-block) * time.Second)
		if queryBegin.Before(beginTime) {
			querySec = int(queryEnd.Sub(beginTime).Seconds())
			queryBegin = beginTime
		}
		startTime0 := time.Now()

		qInfo.queryRangeInfo = queryRangeInfo{
			queryBegin:  queryBegin,
			queryEnd:    queryEnd,
			intervalSec: querySec,
		}
		logInfo := ""
		if concurrency == 1 {
			if err := collectSingleQuery(l, c, curTokenID, resultDir, mtc, nameSuffix, qInfo); err != nil {
				l.Errorf("Error quering metrics %s: %s... timeout:%v, take time:%v",
					mtc+nameSuffix, err, c.Timeout*3+5*time.Second, time.Since(startTime0))
			}
		} else {
			queryInfoCh <- qInfo
			if goCnt == 0 {
				wg.RunWithRecover(func() { collectQueries(l, c, curTokenID, resultDir, mtc, nameSuffix, queryInfoCh) }, nil)
			} else if goCnt < concurrency {
				token := tl.TryGet()
				if token != nil {
					logInfo = fmt.Sprintf(" with a new goroutine ID:%v", token.ID)
					wg.RunWithRecover(func() {
						collectQueries(l, c, token.ID, resultDir, mtc, nameSuffix, queryInfoCh)
						tl.Put(token)
					}, nil)
				} else {
					logInfo = " try get failed"
				}
			}
			l.Infof("Collecting single metric %s%s, interval:%d s, put task no.%d range[%v:%v] to chan ...",
				mtc+nameSuffix, logInfo, qInfo.intervalSec, goCnt, queryBegin.Format(time.RFC3339), queryEnd.Format(time.RFC3339))
		}
		goCnt++
	}
	if concurrency == 1 {
		return
	}

	startTime1 := time.Now()
	for {
		if len(queryInfoCh) == 0 {
			close(queryInfoCh)
			break
		}
		if wg.PanicCnt == concurrency {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	wg.Wait()
	l.Infof("Collected single metric %s from %s to %s take time:%v, total task:%v, concurrency:%d, wait take time:%v",
		mtc+nameSuffix, endTime.Format(time.RFC3339), beginTime.Format(time.RFC3339), time.Since(startTime), goCnt, concurrency, time.Since(startTime1))
}

type queryInfo struct {
	query    string
	promAddr string
	queryRangeInfo
	customHeader []string
	compress     bool
	retryOption  tiuputils.RetryOption
}

type queryRangeInfo struct {
	queryBegin  time.Time
	queryEnd    time.Time
	intervalSec int
}

func collectQueries(l *logprinter.Logger, c *http.Client, tokenID int, resultDir, mtc, nameSuffix string,
	queryInfoCh chan queryInfo) {
	for {
		qInfo, ok := <-queryInfoCh
		if !ok {
			l.Infof("[ID:%d] collect metric %s goroutine finished", tokenID, mtc+nameSuffix)
			return
		}

		startTime0 := time.Now()
		err := collectSingleQuery(l, c, tokenID, resultDir, mtc, nameSuffix, qInfo)
		if err != nil {
			l.Errorf("[ID:%d] failed retry collecting a query metric %s: %s... client timeout:%v, take time:%v",
				tokenID, mtc+nameSuffix, err, c.Timeout*3+5*time.Second, time.Since(startTime0))
		}
	}
}

func collectSingleQuery(l *logprinter.Logger, c *http.Client, tokenID int, resultDir, mtc, nameSuffix string, qInfo queryInfo) error {
	i := 0
	return tiuputils.Retry(
		func() error {
			startTime := time.Now()
			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://%s/api/v1/query?%s", qInfo.promAddr, url.Values{
					"query": {fmt.Sprintf("%s[%ds]", qInfo.query, qInfo.intervalSec)},
					"time":  {qInfo.queryEnd.Format(time.RFC3339)},
				}.Encode()),
				nil)
			if err != nil {
				return err
			}
			getTime := time.Since(startTime)
			utils.AddHeaders(req.Header, qInfo.customHeader)
			resp, err := c.Do(req)
			i++
			if err != nil {
				l.Errorf("[ID:%d-try:%d] failed query metric %s: %s retry... interval:%v s, take time:%v. If prometheus OOM is the cause, consider reducing concurrency and metrics-min-interval",
					tokenID, i, mtc+nameSuffix, err, qInfo.intervalSec, getTime)
				time.Sleep(200 * time.Millisecond)
				return err
			}
			// Prometheus API response format is JSON. Every successful API request returns a 2xx status code.
			if resp.StatusCode/100 != 2 {
				l.Errorf("[ID:%d-try:%d] failed query metric %s Status Code %d, retry... interval:%d s, take time:%v",
					tokenID, i, mtc+nameSuffix, resp.StatusCode, qInfo.intervalSec, getTime)
				time.Sleep(200 * time.Millisecond)
			}
			defer resp.Body.Close()

			dst, err := os.Create(
				filepath.Join(
					resultDir, subdirMonitor, subdirMetrics, strings.ReplaceAll(qInfo.promAddr, ":", "-"),
					fmt.Sprintf("%s-%s-%s%s.json", mtc, qInfo.queryBegin.Format(time.RFC3339), qInfo.queryEnd.Format(time.RFC3339), nameSuffix),
				),
			)
			if err != nil {
				l.Errorf("[ID:%d-try:%d] failed query metric %s: %s, retry...", tokenID, i, mtc+nameSuffix, err)
			}
			defer dst.Close()

			var enc io.WriteCloser
			var n int64
			if qInfo.compress {
				// compress the metric
				enc, err = zstd.NewWriter(dst)
				if err != nil {
					l.Errorf("[ID:%d-try:%d] failed compressing metric %s: %s, retry...\n", tokenID, i, mtc+nameSuffix, err)
					return err
				}
				defer enc.Close()
			} else {
				enc = dst
			}
			n, err = io.Copy(enc, resp.Body)
			if err != nil {
				l.Errorf("[ID:%d-try:%d] failed writing metric err %s to file: %s, retry...take time:%v \n",
					tokenID, i, mtc+nameSuffix, err, time.Since(startTime))
				return err
			}
			if time.Since(startTime) > time.Second {
				l.Infof("[ID:%d-try:%d] Collected a query metric %s from %s to %s (%d bytes) take a long time:%v",
					tokenID, i, mtc+nameSuffix, qInfo.queryBegin.Format(time.RFC3339), qInfo.queryEnd.Format(time.RFC3339), n, time.Since(startTime))
			}
			return nil
		},
		qInfo.retryOption,
	)
}

func ensureMonitorDir(base string, sub ...string) error {
	e := []string{base, subdirMonitor}
	e = append(e, sub...)
	dir := path.Join(e...)
	return os.MkdirAll(dir, 0755)
}

func filterMetrics(src, filter, exclude []string) []string {
	var res []string
	for _, metric := range src {
		if (len(filter) < 1 || utils.MatchPrefixs(metric, filter)) &&
			!utils.MatchPrefixs(metric, exclude) {
			res = append(res, metric)
		}
	}

	return res
}

func generateQueryWitLabel(metric string, labels map[string]string) string {
	buf := new(strings.Builder)
	buf.WriteString("{")
	fmt.Fprintf(buf, "__name__=%q", metric)
	for name, value := range labels {
		fmt.Fprintf(buf, ",%s=%q", name, value)
	}
	buf.WriteString("}")
	return buf.String()
}

// TSDBCollectOptions is the options collecting TSDB file of prometheus, only work for tiup-cluster deployed cluster
type TSDBCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	resultDir string
	fileStats map[string][]CollectStat
	compress  bool
	limit     int
}

// Desc implements the Collector interface
func (c *TSDBCollectOptions) Desc() string {
	return "metrics from Prometheus node"
}

// GetBaseOptions implements the Collector interface
func (c *TSDBCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *TSDBCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *TSDBCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *TSDBCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *TSDBCollectOptions) Prepare(m *Manager, cls *models.TiDBCluster) (map[string][]CollectStat, error) {
	if m.mode != CollectModeTiUP {
		return nil, nil
	}
	if len(cls.Monitors) < 1 {
		m.logger.Warnf("No Prometheus node found in topology, skip.")
		return nil, nil
	}

	// tsEnd, _ := utils.ParseTime(c.GetBaseOptions().ScrapeEnd)
	// tsStart, _ := utils.ParseTime(c.GetBaseOptions().ScrapeBegin)

	uniqueHosts := map[string]int{}             // host -> ssh-port
	uniqueArchList := make(map[string]struct{}) // map["os-arch"]{}
	hostPaths := make(map[string]set.StringSet)
	hostTasks := make(map[string]*task.Builder)

	topo := cls.Attributes[CollectModeTiUP].(spec.Topology)
	components := topo.ComponentsByStartOrder()
	var (
		dryRunTasks   []*task.StepDisplay
		downloadTasks []*task.StepDisplay
	)

	for _, comp := range components {
		if comp.Name() != spec.ComponentPrometheus {
			continue
		}

		for _, inst := range comp.Instances() {
			archKey := fmt.Sprintf("%s-%s", inst.OS(), inst.Arch())
			if _, found := uniqueArchList[archKey]; !found {
				uniqueArchList[archKey] = struct{}{}
				t0 := task.NewBuilder(m.logger).
					Download(
						componentDiagCollector,
						inst.OS(),
						inst.Arch(),
						"", // latest version
					).
					BuildAsStep(fmt.Sprintf("  - Downloading collecting tools for %s/%s", inst.OS(), inst.Arch()))
				downloadTasks = append(downloadTasks, t0)
			}

			// tasks that applies to each host
			if _, found := uniqueHosts[inst.GetHost()]; !found {
				uniqueHosts[inst.GetHost()] = inst.GetSSHPort()
				// build system info collecting tasks
				t1, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
				if err != nil {
					return nil, err
				}
				t1 = t1.
					Mkdir(c.GetBaseOptions().User, inst.GetHost(), filepath.Join(task.CheckToolsPathDir, "bin")).
					CopyComponent(
						componentDiagCollector,
						inst.OS(),
						inst.Arch(),
						"", // latest version
						"", // use default srcPath
						inst.GetHost(),
						task.CheckToolsPathDir,
					)
				hostTasks[inst.GetHost()] = t1
			}

			// add filepaths to list
			if _, found := hostPaths[inst.GetHost()]; !found {
				hostPaths[inst.GetHost()] = set.NewStringSet()
			}
			hostPaths[inst.GetHost()].Insert(inst.DataDir())
		}
	}

	// build scraper tasks
	for h, t := range hostTasks {
		host := h
		t = t.
			Shell(
				host,
				fmt.Sprintf("%s --prometheus '%s' -f '%s' -t '%s'",
					filepath.Join(task.CheckToolsPathDir, "bin", "scraper"),
					strings.Join(hostPaths[host].Slice(), ","),
					c.ScrapeBegin, c.ScrapeEnd,
				),
				"",
				false,
			).
			Func(
				host,
				func(ctx context.Context) error {
					stats, err := parseScraperSamples(ctx, host)
					if err != nil {
						return err
					}
					for host, files := range stats {
						c.fileStats[host] = files
					}
					return nil
				},
			)
		t1 := t.BuildAsStep(fmt.Sprintf("  - Scraping prometheus data files on %s:%d", host, uniqueHosts[host]))
		dryRunTasks = append(dryRunTasks, t1)
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Download necessary tools", false, downloadTasks...).
		ParallelStep("+ Collect host information", false, dryRunTasks...).
		Build()

	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)
	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return nil, err
		}
		return nil, perrs.Trace(err)
	}

	return c.fileStats, nil
}

// Collect implements the Collector interface
func (c *TSDBCollectOptions) Collect(m *Manager, cls *models.TiDBCluster) error {
	if m.mode != CollectModeTiUP {
		return nil
	}

	topo := cls.Attributes[CollectModeTiUP].(spec.Topology)
	var (
		collectTasks []*task.StepDisplay
		cleanTasks   []*task.StepDisplay
	)
	uniqueHosts := map[string]int{} // host -> ssh-port

	components := topo.ComponentsByStartOrder()

	for _, comp := range components {
		if comp.Name() != spec.ComponentPrometheus {
			continue
		}

		insts := comp.Instances()
		if len(insts) < 1 {
			return nil
		}

		// only collect from first promethes
		inst := insts[0]
		// checks that applies to each host
		if _, found := uniqueHosts[inst.GetHost()]; found {
			continue
		}
		uniqueHosts[inst.GetHost()] = inst.GetSSHPort()

		t2, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
		if err != nil {
			return err
		}
		for _, f := range c.fileStats[inst.GetHost()] {
			// build checking tasks
			t2 = t2.
				// check for listening ports
				CopyFile(
					f.Target,
					filepath.Join(c.resultDir, subdirMonitor, subdirRaw, fmt.Sprintf("%s-%d", inst.GetHost(), inst.GetMainPort()), filepath.Base(f.Target)),
					inst.GetHost(),
					true,
					c.limit,
					c.compress,
				)
		}
		collectTasks = append(
			collectTasks,
			t2.BuildAsStep(fmt.Sprintf("  - Downloading prometheus data files from node %s", inst.GetHost())),
		)

		b, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
		if err != nil {
			return err
		}
		t3 := b.
			Rmdir(inst.GetHost(), task.CheckToolsPathDir).
			BuildAsStep(fmt.Sprintf("  - Cleanup temp files on %s:%d", inst.GetHost(), inst.GetSSHPort()))
		cleanTasks = append(cleanTasks, t3)
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Scrap files on nodes", false, collectTasks...).
		ParallelStep("+ Cleanup temp files", false, cleanTasks...).
		Build()

	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)
	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

func (c *MetricCollectOptions) NewForwardPorts(podName string, port int) (chan struct{}, int, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", c.Kubeconfig)
	if err != nil {
		return nil, 0, err
	}

	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get kubernetes Clientset: %v", err)
	}

	req := kubeCli.CoreV1().RESTClient().Post().Namespace(c.MonitorNamespace).
		Resource("pods").Name(podName).SubResource("portforward")

	var readyChannel, stopChannel chan struct{}
	readyChannel = make(chan struct{})
	stopChannel = make(chan struct{}, 1)
	transport, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return nil, 0, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	fw, err := portforward.NewOnAddresses(dialer, []string{"127.0.0.1"}, []string{fmt.Sprintf(":%d", port)}, stopChannel, readyChannel, nil, nil)
	if err != nil {
		return nil, 0, err
	}
	go fw.ForwardPorts()
	<-readyChannel

	ports, err := fw.GetPorts()
	if err != nil {
		return nil, 0, err
	}

	return stopChannel, int(ports[0].Local), nil
}
