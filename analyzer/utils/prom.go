package utils

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	pm "github.com/prometheus/common/model"
)

func QueryProm(query string, t time.Time) (*float64, error) {
	addr := os.Getenv("PROM_ADDR")
	if addr == "" {
		addr = "http://127.0.0.1:8080"
	}
	client, err := api.NewClient(api.Config{Address: addr})
	if err != nil {
		return nil, err
	}
	api := prom.NewAPI(client)
	v, _, err := api.Query(context.Background(), query, t)
	if err != nil {
		return nil, err
	}

	vec, ok := v.(pm.Vector)
	if !ok {
		return nil, errors.New("query prometheus: result type mismatch")
	}

	if len(vec) == 0 {
		return nil, errors.New("metric not found")
	}

	value := float64(vec[0].Value)
	return &value, nil
}

func QueryPromRange(query string, start, end time.Time, step time.Duration) (FloatArray, error) {
	values := FloatArray{}
	addr := os.Getenv("PROM_ADDR")
	if addr == "" {
		addr = "http://127.0.0.1:8080"
	}
	client, err := api.NewClient(api.Config{Address: addr})
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

type FloatArray []float64

func (array FloatArray) Max() float64 {
	if len(array) == 0 {
		return 0
	}
	max := array[0]
	for _, v := range array[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func (array FloatArray) Min() float64 {
	if len(array) == 0 {
		return 0
	}
	min := array[0]
	for _, v := range array[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func (array FloatArray) Avg() float64 {
	if len(array) == 0 {
		return 0
	}
	var sum float64 = 0
	for _, v := range array[1:] {
		sum += v
	}
	return sum / float64(len(array))
}
