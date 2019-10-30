package emphasis

import (
	"database/sql"
	"github.com/pingcap/tidb-foresight/utils"
)

// The Problem in Emphasis.
type Problem struct {
	CreateTime   utils.NullTime `json:"create_time"`
	InspectionId string         `json:"inspection_id"`
	Uuid         string         `json:"uuid" gorm:"PRIMARY_KEY"`
	RelatedGraph string         `json:"related_graph"` // Related Grafana Graph.
	Problem      sql.NullString `json:"problem"`       // Related problem, return json null to represent no problem here.
	Advise       string         `json:"advise"`
}
