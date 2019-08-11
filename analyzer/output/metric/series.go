package metric

// seriesT is a stream of data points belonging to a metric.
type seriesT struct {
	Metric map[string]string `json:"metric"`
	Points []pointT          `json:"values"`
}
