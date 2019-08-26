package parser

import (
	"time"

	"github.com/pingcap/tidb-foresight/log/item"
)

// The log parser representation
type Parser interface {
	// Parse the first line of the log, and return ts and level,
	// if it's not the first line, nil will be returned.
	ParseHead(head []byte) (*time.Time, item.LevelType)
}

// All parsers this package has
func List() []Parser {
	return []Parser{
		&PDLogV2Parser{},
		&TiKVLogV2Parser{},
		&UnifiedLogParser{},
		&SlowQueryParser{},
	}
}
