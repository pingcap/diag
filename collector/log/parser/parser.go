package parser

import (
	"time"

	"github.com/pingcap/diag/collector/log/item"
)

// Parser is the log parser representation
type Parser interface {
	// Parse the first line of the log, and return ts and level,
	// if it's not the first line, nil will be returned.
	ParseHead(head []byte) (*time.Time, item.LevelType)
}

// List all parsers this package has
// **not used anymore**
func List() []Parser {
	return []Parser{
		&UnifiedJSONLogParser{},
		&UnifiedLogParser{},
		&SlowQueryParser{},
		&TiFlashErrLogParser{},
		&PDLogV2Parser{},
		&TiKVLogV2Parser{},
	}
}

// ListStd return all std log parsers this package has
func ListStd() []Parser {
	return []Parser{
		&UnifiedJSONLogParser{},
		&UnifiedLogParser{},
		&TiFlashErrLogParser{},
		&PrometheusLogParser{},
		&PDLogV2Parser{},
		&TiKVLogV2Parser{},
		&AuditJSONLogParser{},
	}
}
