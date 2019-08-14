package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type LevelType int16

const (
	LevelFATAL LevelType = iota
	LevelERROR
	LevelWARN
	LevelINFO
	LevelDEBUG
)

const (
	TimeStampLayout       = "2006/01/02 15:04:05.000 -07:00"
	FormerTimeStampLayout = "2006/01/02 15:04:05.000"
)

var (
	UnifiedLog = regexp.MustCompile(`^\[([^\[\]]*)\]\s\[([^\[\]]*)\]`)
	TiKVLogRE  = regexp.MustCompile(`^([0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3})\s([^\s]*)`)
	PDLogRE    = regexp.MustCompile(`^([0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3})\s[^\s]*\s\[([^\[\]]*)\]`)
	SlowLogRE  = regexp.MustCompile("^# Time: (.*)$")
)

type ParseFunc func(line []byte) (*time.Time, LevelType, error)

func parseUnifiedLog(line []byte) (*time.Time, LevelType, error) {
	if !UnifiedLog.Match(line) {
		return nil, -1, fmt.Errorf("skip parse illegal log line: %s", line)
	}
	matches := UnifiedLog.FindSubmatch(line)
	t, err := parseTimeStamp(matches[1])
	if err != nil {
		return nil, -1, err
	}
	level, err := parseLogLevel(matches[2])
	if err != nil {
		return nil, -1, err
	}
	return t, level, nil
}

func parseTiKVLog(line []byte) (*time.Time, LevelType, error) {
	if !TiKVLogRE.Match(line) {
		return nil, -1, fmt.Errorf("skip parse illegal TiKV log line: %s", line)
	}
	matches := TiKVLogRE.FindSubmatch(line)
	t, err := parseTimeStamp(matches[1])
	if err != nil {
		return nil, -1, err
	}
	level, err := parseLogLevel(matches[2])
	if err != nil {
		return nil, -1, err
	}
	return t, level, nil
}

func parsePDLog(line []byte) (*time.Time, LevelType, error) {
	if !PDLogRE.Match(line) {
		return nil, -1, fmt.Errorf("skip parse illegal PD log line: %s", line)
	}
	matches := PDLogRE.FindSubmatch(line)
	t, err := parseFormerTimeStamp(matches[1])
	if err != nil {
		return nil, -1, err
	}
	level, err := parseLogLevel(matches[2])
	if err != nil {
		return nil, -1, err
	}
	return t, level, nil
}

// TiDB / TiKV / PD unified log format
// [2019/03/04 17:04:24.614 +08:00] ...
func parseTimeStamp(b []byte) (*time.Time, error) {
	t, err := time.Parse(TimeStampLayout, string(b))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TiDB / TiKV / PD log format used in former version
// 2019/07/18 11:04:29.314 ...
func parseFormerTimeStamp(b []byte) (*time.Time, error) {
	local, err := time.LoadLocation("Asia/Chongqing")
	if err != nil {
		return nil, err
	}
	t, err := time.ParseInLocation(FormerTimeStampLayout, string(b), local)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

var LevelTypeMap = map[string]LevelType{
	"FATAL": LevelFATAL,
	"ERROR": LevelERROR,
	"WARN":  LevelWARN,
	"INFO":  LevelINFO,
	"DEBUG": LevelDEBUG,
}

func parseLogLevel(b []byte) (LevelType, error) {
	s := strings.ToUpper(string(b))
	if s == "ERRO" {
		return LevelERROR, nil
	}
	if s == "CRITICAL" {
		return LevelFATAL, nil
	}
	if s == "WARNING" {
		return LevelWARN, nil
	}
	if level, ok := LevelTypeMap[s]; ok {
		return level, nil
	} else {
		return -1, fmt.Errorf("failed to parse log level: %s", s)
	}
}
