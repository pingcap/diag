package meta

import (
	"time"
)

// The Meta struct represent the basic inspection information
type Meta struct {
	// When the collector start collect this inspection information
	InspectTime time.Time `json:"inspect_time"`
	// When the collector end collect this inspection information
	EndTime time.Time `json:"end_time"`
	// How many tidb alive in the metric time range
	TidbCount int `json:"tidb_count"`
	// How many tikv alive in the metric time range
	TikvCount int `json:"tikv_count"`
	// How many pd alive in the metric time range
	PdCount int `json:"pd_count"`
}
