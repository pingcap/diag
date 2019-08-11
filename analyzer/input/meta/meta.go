package meta

// The Meta struct represent the basic inspection information
type Meta struct {
	// The instance id unique identity a cluster, which owns the inspection
	InstanceId string `json:"instance_id"`
	// The cluster name of which the inspection belongs to
	ClusterName string `json:"cluster_name"`
	// The inspection create time
	CreateTime float64 `json:"create_time"`
	// When the collector start collect this inspection information
	InspectTime float64 `json:"inspect_time"`
	// When the collector end colledt this inspection information
	EndTime float64 `json:"end_time"`
	// How many tidb alive in the metric time range
	TidbCount int `json:"tidb_count"`
	// How many tikv alive in the metric time range
	TikvCount int `json:"tikv_count"`
	// How many pd alive in the metric time range
	PdCount int `json:"pd_count"`
}
