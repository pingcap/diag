package parser

import (
	"regexp"
	"time"

	"github.com/pingcap/diag/log/item"
)

// Parse pd (v2.x) log, example:
// 2019/08/21 02:11:54.405 util.go:59: [info] Release Version: v2.1.12
type PDLogV2Parser struct{}

var (
	PDLogV2RE = regexp.MustCompile(`^([0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3})\s[^\s]*\s\[([^\[\]]*)\]`)
)

func (*PDLogV2Parser) ParseHead(head []byte) (*time.Time, item.LevelType) {
	if !PDLogV2RE.Match(head) {
		return nil, item.LevelInvalid
	}
	matches := PDLogV2RE.FindSubmatch(head)
	t, err := parseFormerTimeStamp(matches[1])
	if err != nil {
		return nil, item.LevelInvalid
	}
	level := ParseLogLevel(matches[2])
	if level == item.LevelInvalid {
		return nil, item.LevelInvalid
	}
	return t, level
}
