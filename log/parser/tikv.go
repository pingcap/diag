package parser

import (
	"regexp"
	"time"

	"github.com/pingcap/diag/log/item"
)

// Parse tikv (v2.x) log, example:
// 2019/08/13 03:27:49.382 INFO mod.rs:26: Welcome to TiKV.
// Release Version:   2.1.13
// Git Commit Hash:   b3cf3c8d642534ea6fa93d475a46da285cc6acbf
// Git Commit Branch: HEAD
// UTC Build Time:    2019-06-21 12:25:23
// Rust Version:      rustc 1.29.0-nightly (4f3c7a472 2018-07-17)
type TiKVLogV2Parser struct{}

var (
	TiKVLogRE = regexp.MustCompile(`^([0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3})\s([^\s]*)`)
)

func (*TiKVLogV2Parser) ParseHead(head []byte) (*time.Time, item.LevelType) {
	if !TiKVLogRE.Match(head) {
		return nil, item.LevelInvalid
	}
	matches := TiKVLogRE.FindSubmatch(head)
	t, err := parseTimeStamp(matches[1])
	if err != nil {
		return nil, item.LevelInvalid
	}
	level := ParseLogLevel(matches[2])
	if level == item.LevelInvalid {
		return nil, item.LevelInvalid
	}
	return t, level
}
