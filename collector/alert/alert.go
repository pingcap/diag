package alert

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
)

type Options interface {
	GetHome() string
	GetInspectionId() string
	GetPrometheusEndpoint() (string, error)
}

type AlertCollector struct {
	opts Options
}

func New(opts Options) *AlertCollector {
	return &AlertCollector{opts}
}

func (b *AlertCollector) Collect() error {
	home := b.opts.GetHome()
	inspection := b.opts.GetInspectionId()

	promAddr, err := b.opts.GetPrometheusEndpoint()
	if err != nil {
		return err
	}

	resp, err := http.PostForm(fmt.Sprintf("http://%s/api/v1/query", promAddr), url.Values{"query": {"ALERTS"}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dst, err := os.Create(path.Join(home, "inspection", inspection, "alert.json"))
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, resp.Body)
	return err
}
