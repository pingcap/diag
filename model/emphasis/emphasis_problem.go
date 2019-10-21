package emphasis

// 重点问题中排查的问题选项
type Problem struct {
	Uuid string `json:"uuid" gorm:"PRIMARY_KEY"`
	// 相关的 Grafana 监控图
	RelatedGraph string `json:"related_graph"`
	// 具体的问题, 返回 json null 代表没问题
	Problem string `json:"problem"`
}
