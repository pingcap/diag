package alert

// The alert infomation from prometheus
type Alert []struct {
	Metric struct {
		Name string `json:"alertname"`
	} `json:"metric"`
	Value []interface{}
}
