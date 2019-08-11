package args

import (
	"time"
)

// The args foresight api passed to collector
// Is parsed from args.json
type Args struct {
	InstanceId   string    `json:"instance_id"`
	InspectionId string    `json:"inspection_id"`
	Collects     string    `json:"collect"`
	ScrapeBegin  time.Time `json:"begin"`
	ScrapeEnd    time.Time `json:"end"`
}
