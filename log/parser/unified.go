package parser

import (
	"regexp"
	"time"

	"github.com/pingcap/tidb-foresight/log/item"
)

// Parse unified log, include tidb v2 and v3, pd v3 and tikv v3, examples:
// [2019/08/26 06:19:13.011 -04:00] [INFO] [printer.go:41] ["Welcome to TiDB."] ["Release Version"=v2.1.14]...
// [2019/08/26 07:19:49.529 -04:00] [INFO] [printer.go:41] ["Welcome to TiDB."] ["Release Version"=v3.0.2]...
// [2019/08/21 01:43:01.460 -04:00] [INFO] [util.go:60] [PD] [release-version=v3.0.2]
// [2019/08/26 07:20:23.815 -04:00] [INFO] [mod.rs:28] ["Release Version:   3.0.2"]
type UnifiedLogParser struct{}

var (
	UnifiedLogRE = regexp.MustCompile(`^\[([^\[\]]*)\]\s\[([^\[\]]*)\]`)
)

func (*UnifiedLogParser) ParseHead(head []byte) (*time.Time, item.LevelType) {
	if !UnifiedLogRE.Match(head) {
		return nil, item.LevelInvalid
	}
	matches := UnifiedLogRE.FindSubmatch(head)
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
