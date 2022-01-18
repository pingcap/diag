package parser

import (
	"time"

	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/collector/log/item"
)

// Parse unified JSON log, include tidb v2 and v3, pd v3 and tikv v3, examples:
// {"level":"INFO","time":"2022/01/14 08:09:55.307 +01:00","caller":"printer.go:34","message":"Welcome to TiDB.","Release Version":"v5.3.0","Edition":"Community","Git Commit Hash":"4a1b2e9fe5b5afb1068c56de47adb07098d768d6","Git Branch":"heads/refs/tags/v5.3.0","UTC Build Time":"2021-11-24 13:32:39","GoVersion":"go1.16.4","Race Enabled":false,"Check Table Before Drop":false,"TiKV Min Version":"v3.0.0-60965b006877ca7234adaced7890d7b029ed1306"}
type UnifiedJSONLogParser struct{}

func (*UnifiedJSONLogParser) ParseHead(head []byte) (*time.Time, item.LevelType) {
	if !json.Valid(head) {
		return nil, item.LevelInvalid
	}

	var headmap map[string]interface{}
	if err := json.Unmarshal(head, &headmap); err != nil {
		return nil, item.LevelInvalid
	}

	entryTime, ok := headmap["time"].(string)
	if !ok {
		return nil, item.LevelInvalid
	}
	t, err := parseTimeStamp([]byte(entryTime))
	if err != nil {
		return nil, item.LevelInvalid
	}

	entryLevel, ok := headmap["level"].(string)
	if !ok {
		return nil, item.LevelInvalid
	}
	level := ParseLogLevel([]byte(entryLevel))
	if level == item.LevelInvalid {
		return nil, item.LevelInvalid
	}
	return t, level
}
