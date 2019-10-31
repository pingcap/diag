package metric

import (
	"github.com/pingcap/tidb-foresight/wrapper/prometheus"
)

// metricT represents a metric returned from the prometheus api
type metricT struct {
	ResultType string  `json:"resultType"`
	Result     matrixT `json:"result"`
}

// This is used for other tasks for dependency reason.
// It's empty because other tasks should query prometheus
// for result, this is just to make sure this taks is executed
// before other tasks dependened on it.
type Metric struct {
	prometheus.Prometheus
}
