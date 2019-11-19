package metric

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

// The prometheus can give at most 11000 points for every series.
// So, for precision you can set MAX_POINTS with a higher value
// but no more than 11000.
const MAX_POINTS = 11000

type Options interface {
	GetHome() string
	GetInspectionId() string
	GetScrapeBegin() (time.Time, error)
	GetScrapeEnd() (time.Time, error)
	GetPrometheusEndpoint() (string, error)
}

type MetricCollector struct {
	opts Options
}

func New(opts Options) *MetricCollector {
	return &MetricCollector{opts}
}

func (m *MetricCollector) Collect() error {
	home := m.opts.GetHome()
	inspection := m.opts.GetInspectionId()

	end, err := m.opts.GetScrapeEnd()
	if err != nil {
		end = time.Now()
	}
	begin, err := m.opts.GetScrapeBegin()
	if err != nil {
		begin = end.Add(time.Duration(-1) * time.Hour)
	}
	promAddr, err := m.opts.GetPrometheusEndpoint()
	if err != nil {
		return err
	}
	metrics, err := m.getMetricList(promAddr)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(path.Join(home, "inspection", inspection, "metric"), os.ModePerm); err != nil {
		return err
	}

	tl := utils.NewTokenLimiter(uint(runtime.NumCPU()))
	for _, mtc := range metrics {
		go func(tok *utils.Token) {
			m.collectMetric(promAddr, begin, end, mtc)
			tl.Put(tok)
		}(tl.Get())
	}
	tl.Wait()

	return nil
}

func (m *MetricCollector) getMetricList(prom string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/label/__name__/values", prom))
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

func (m *MetricCollector) collectMetric(prom string, begin, end time.Time, mtc string) {
	home := m.opts.GetHome()
	inspection := m.opts.GetInspectionId()

	duration := end.Sub(begin)
	step := int(duration.Seconds()/MAX_POINTS + 1)
	if step < 15 { // the most accurate prometheus can give (15s a point)
		step = 15
	}

	resp, err := http.PostForm(
		fmt.Sprintf("http://%s/api/v1/query_range", prom),
		url.Values{
			"query": {mtc},
			"start": {begin.Format(time.RFC3339)},
			"end":   {end.Format(time.RFC3339)},
			"step":  {strconv.Itoa(step)},
		},
	)
	if err != nil {
		log.Errorf("collect metric %s: %s", mtc, err)
		return
	}
	defer resp.Body.Close()

	dst, err := os.Create(path.Join(
		home, "inspection", inspection, "metric",
		fmt.Sprintf("%s_%s_%s.json", mtc, begin.Format(time.RFC3339), end.Format(time.RFC3339)),
	))
	if err != nil {
		log.Errorf("collect metric %s: %s", mtc, err)
		return
	}
	defer dst.Close()

	if _, err = io.Copy(dst, resp.Body); err != nil {
		log.Errorf("collect metric %s: %s", mtc, err)
		return
	}
}
