package parser

import (
	"regexp"
	"time"

	"github.com/pingcap/diag/collector/log/item"
)

// Parse prometheus and alertmanager log, example:
// level=warn ts=2022-06-09T09:16:54.674Z caller=main.go:377 deprecation_notice="'storage.tsdb.retention' flag is deprecated use 'storage.tsdb.retention.time' instead."
type PrometheusLogParser struct{}

var (
	PrometheusLogRE = regexp.MustCompile(`^level=([a-z]*) ts=([0-9]{4}\-[0-9]{2}\-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}[\S]*) [.]*`)
)

func (*PrometheusLogParser) ParseHead(head []byte) (*time.Time, item.LevelType) {
	if !PrometheusLogRE.Match(head) {
		return nil, item.LevelInvalid
	}
	matches := PrometheusLogRE.FindSubmatch(head)
	t, err := parsePromTimeStamp(matches[2])
	if err != nil {
		return nil, item.LevelInvalid
	}
	level := ParseLogLevel(matches[1])
	if level == item.LevelInvalid {
		return nil, item.LevelInvalid
	}
	return t, level
}
