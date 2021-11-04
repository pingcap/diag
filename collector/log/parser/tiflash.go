package parser

import (
	"regexp"
	"time"

	"github.com/pingcap/diag/collector/log/item"
)

// Parse tiflash error log, example:
// 2021.11.01 23:31:29.123456 [ 1 ] <Error> pingcap.tikv: Get Failed14: failed to connect to all addresses
type TiFlashErrLogParser struct{}

var (
	TiFlashErrLogRE = regexp.MustCompile(`^([0-9]{4}\.[0-9]{2}\.[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}) \[ [0-9] \] <([^\[\]]*)>`)
)

func (*TiFlashErrLogParser) ParseHead(head []byte) (*time.Time, item.LevelType) {
	if !TiFlashErrLogRE.Match(head) {
		return nil, item.LevelInvalid
	}
	matches := TiFlashErrLogRE.FindSubmatch(head)
	t, err := parseTiFlashErrTimeStamp(matches[1])
	if err != nil {
		return nil, item.LevelInvalid
	}
	level := ParseLogLevel(matches[2])
	if level == item.LevelInvalid {
		return nil, item.LevelInvalid
	}
	return t, level
}
