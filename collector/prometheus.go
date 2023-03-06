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
	"archive/tar"
	"bytes"
	"context"
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

	"github.com/joomcode/errorx"
	json "github.com/json-iterator/go"
	"github.com/klauspost/compress/zstd"
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/diag/scraper"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/tui/progress"
	tiuputils "github.com/pingcap/tiup/pkg/utils"
	k8sutils "github.com/qiffang/k8sutils/pkg/exec"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	subdirMonitor = "monitor"
	subdirAlerts  = "alerts"
	subdirMetrics = "metrics"
	subdirRaw     = "raw"
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
	opt       *operator.Options // global operations from cli
	resultDir string
	label     map[string]string
	metrics   []string // metric list
	filter    []string
	limit     int // series*min per query
	compress  bool
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
				collectMetric(m.logger, client, prom, tsStart, tsEnd, mtc, c.label, c.resultDir, c.limit, c.compress)

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

func getSeriesNum(c *http.Client, promAddr, query string) (int, error) {
	resp, err := c.Get(
		fmt.Sprintf("http://%s/api/v1/series?match[]=%s", promAddr, query),
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
	mtc string,
	label map[string]string,
	resultDir string,
	speedlimit int,
	compress bool,
) {
	query := generateQueryWitLabel(mtc, label)
	l.Debugf("Querying series of %s...", mtc)
	series, err := getSeriesNum(c, promAddr, query)
	if err != nil {
		l.Errorf("%s", err)
		return
	}
	if series <= 0 {
		l.Debugf("metric %s has %d series, ignore", mtc, series)
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
						"query": {fmt.Sprintf("%s[%ds]", query, querySec)},
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

				var enc io.WriteCloser
				var n int64
				if compress {
					// compress the metric
					enc, err = zstd.NewWriter(dst)
					if err != nil {
						l.Errorf("failed compressing metric %s: %s, retry...\n", mtc, err)
						return err
					}
					defer enc.Close()
				} else {
					enc = dst
				}
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

func generateQueryWitLabel(metric string, labels map[string]string) string {
	query := metric
	if len(labels) > 0 {
		query += "{"
		for k, v := range labels {
			query = fmt.Sprintf("%s%s=\"%s\",", query, k, v)
		}
		query = query[:len(query)-1] + "}"
	}
	return query
}

// TSDBCollectOptions is the options collecting TSDB file of prometheus, only work for tiup-cluster deployed cluster
type TSDBCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	resultDir string
	fileStats map[string][]CollectStat
	compress  bool
	limit     int
	fCli      *k8sutils.Client
	pod       string
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
		return c.kubePrepare(m, cls)
	}
	if len(cls.Monitors) < 1 {
		if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
			fmt.Println("No Prometheus node found in topology, skip.")
		} else {
			m.logger.Warnf("No Prometheus node found in topology, skip.")
		}
		return nil, nil
	}

	// tsEnd, _ := utils.ParseTime(c.GetBaseOptions().ScrapeEnd)
	// tsStart, _ := utils.ParseTime(c.GetBaseOptions().ScrapeBegin)

	uniqueHosts := map[string]int{}             // host -> ssh-port
	uniqueArchList := make(map[string]struct{}) // map["os-arch"]{}
	hostPaths := make(map[string]set.StringSet)
	hostTasks := make(map[string]*task.Builder)

	topo := cls.Attributes[CollectModeTiUP].(spec.Topology)
	components := topo.ComponentsByUpdateOrder()
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
		return c.kubeCollect(m, cls)
	}

	topo := cls.Attributes[CollectModeTiUP].(spec.Topology)
	var (
		collectTasks []*task.StepDisplay
		cleanTasks   []*task.StepDisplay
	)
	uniqueHosts := map[string]int{} // host -> ssh-port

	components := topo.ComponentsByUpdateOrder()

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

func (c *TSDBCollectOptions) kubePrepare(m *Manager, cls *models.TiDBCluster) (map[string][]CollectStat, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", c.BaseOptions.Kubeconfig)
	if err != nil {
		return nil, err
	}
	c.fCli, err = k8sutils.NewClient(&k8sutils.ClientOpt{
		K8sConfig:     cfg,
		Namespace:     c.Namespace,
		PodName:       c.pod,
		ContainerName: "prometheus",
	})
	if err != nil {
		return nil, err
	}

	datadir := "/data/prometheus"

	err = c.fCli.CanExec()
	if err != nil {
		fmt.Print(err)
		return nil, err
	}

	binpath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	scraperPath := path.Join(path.Dir(binpath), "scraper")

	// todo: copy scraper to pod
	cpCommand := []string{"tar", "--no-same-permissions", "--no-same-owner", "-xmf", "-", "-C", "/tmp"}

	reader, writer := io.Pipe()
	defer reader.Close()

	go func() {
		tarW := tar.NewWriter(writer)
		defer tarW.Close()
		fi, err := os.Stat(scraperPath)
		if err != nil {
			return
		}
		header, _ := tar.FileInfoHeader(fi, "")

		err = tarW.WriteHeader(header)
		if err != nil {
			return
		}

		fd, err := os.Open(scraperPath)
		if err != nil {
			return
		}
		defer fd.Close()

		_, err = io.Copy(tarW, fd)
		if err != nil {
			return
		}

		tarW.Close()
		writer.Close()
	}()

	err = c.fCli.ExecPod(cpCommand, reader, nil, nil, false, 40*(time.Second))
	if err != nil {
		println(err)
	}

	out := &bytes.Buffer{}
	scraperCommand := []string{"/tmp/scraper", "--prometheus", datadir, "-f", c.ScrapeBegin, "-t", c.ScrapeEnd}
	err = c.fCli.ExecPod(scraperCommand, nil, out, nil, false, 40*(time.Second))
	if err != nil {
		println(err)
	}

	stats, err := parseScraperTSDB(out.Bytes())
	if err != nil {
		return nil, err
	}
	c.fileStats[c.pod] = stats

	return c.fileStats, nil
}

func (c *TSDBCollectOptions) kubeCollect(m *Manager, cls *models.TiDBCluster) error {
	for _, f := range c.fileStats[c.pod] {
		downloadCommand := []string{"tar", "cf", "-", "-C", f.Target, "."}
		r, w := io.Pipe()
		go func() {
			utils.Untar(r, filepath.Join(c.resultDir, subdirMonitor, subdirRaw, c.pod, filepath.Base(f.Target)))
		}()
		err := c.fCli.ExecPod(downloadCommand, nil, w, nil, false, 400*(time.Second))
		w.Close()
		if err != nil {
			println(err)
		}
	}

	// f.Target,
	// filepath.Join(c.resultDir, subdirMonitor, subdirRaw, fmt.Sprintf("%s-%d", inst.GetHost(), inst.GetMainPort()), filepath.Base(f.Target)),

	return nil
}

func parseScraperTSDB(stdout []byte) ([]CollectStat, error) {
	var s scraper.Sample
	if err := json.Unmarshal(stdout, &s); err != nil {
		// save output directly on parsing errors
		return nil, fmt.Errorf("error parsing scraped stats: %s", stdout)
	}

	stats := make([]CollectStat, 0)
	for k, v := range s.TSDB {
		stats = append(stats, CollectStat{
			Target: k,
			Size:   v,
		})
	}

	return stats, nil
}
