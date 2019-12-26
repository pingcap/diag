package alert

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/pingcap/tidb-foresight/model"
)

type Options interface {
	GetHome() string
	GetModel() model.Model
	GetInspectionId() string
	GetPrometheusEndpoint() (string, error)
}

type AlertCollector struct {
	Options
}

func New(opts Options) *AlertCollector {
	return &AlertCollector{opts}
}

func (b *AlertCollector) Collect() error {
	home := b.GetHome()
	inspection := b.GetInspectionId()

	b.GetModel().UpdateInspectionMessage(inspection, "collecting alert info...")

	promAddr, err := b.GetPrometheusEndpoint()
	if err != nil {
		return err
	}

	// promAddr
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
