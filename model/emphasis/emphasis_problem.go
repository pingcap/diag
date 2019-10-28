package emphasis

// The Problem in Emphasis.
type Problem struct {
	Uuid string `json:"uuid" gorm:"PRIMARY_KEY"`
	// Related Grafana Graph.
	RelatedGraph string `json:"related_graph"`
	// Related problem, return json null to represent no problem here.
	Problem string `json:"problem"`
}
