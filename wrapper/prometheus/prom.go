package prometheus

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	pm "github.com/prometheus/common/model"
)

type Prometheus interface {
	Query(query string, t time.Time) (float64, error)
	QueryRange(query string, start, end time.Time) (FloatArray, error)
}

func New() Prometheus {
	addr := os.Getenv("PROM_ADDR")
	if addr == "" {
		addr = "http://127.0.0.1:9529"
	}
	return &promeWraper{addr}
}

type promeWraper struct {
	addr string
}

func (p *promeWraper) Query(query string, t time.Time) (float64, error) {
	client, err := api.NewClient(api.Config{Address: p.addr})
	if err != nil {
		return 0, err
	}
	api := prom.NewAPI(client)
	v, _, err := api.Query(context.Background(), query, t)
	if err != nil {
		return 0, err
	}

	vec, ok := v.(pm.Vector)
	if !ok {
		return 0, errors.New("query prometheus: result type mismatch")
	}

	if len(vec) == 0 {
		return 0, errors.New("metric not found")
	}

	value := float64(vec[0].Value)
	return value, nil
}

func (p *promeWraper) QueryRange(query string, start, end time.Time) (FloatArray, error) {
	const points = 11000
	step := end.Sub(start)/points + time.Second
	if step < time.Second*15 {
		step = time.Second * 15
	}
	values := FloatArray{}
	client, err := api.NewClient(api.Config{Address: p.addr})
	if err != nil {
		return values, err
	}
	api := prom.NewAPI(client)
	v, _, err := api.QueryRange(context.Background(), query, prom.Range{start, end, step})
	if err != nil {
		return values, err
	}

	mat, ok := v.(pm.Matrix)
	if !ok {
		return values, errors.New("query prometheus: result type mismatch")
	}

	if len(mat) == 0 {
		return values, nil
	}

	for _, v := range mat[0].Values {
		values = append(values, float64(v.Value))
	}
	return values, nil
}
