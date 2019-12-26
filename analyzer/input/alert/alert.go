package alert

// The alert infomation from prometheus
type Alert []struct {
	Metric struct {
		Name       string `json:"alertname"`
		Alertstate string `json:"alertstate"`
		Env        string `json:"env"`
		Expr       string `json:"expr"`
		Level      string `json:"level"`
		Req        string `json:"req"`
		Instance   string `json:"instance"`

		ExtraInfo map[string]interface{} `json:"-"`
	} `json:"metric"`
	Value []interface{}
}
