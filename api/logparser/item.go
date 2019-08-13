package logparser

import "time"

type Entry struct {
	Type  string // golang type of Value
	Value interface{}
}

type ItemType int16

const (
	TypeTiDB ItemType = iota
	TypeTiKV
	TypePD
	TypeTiDBSlowQuery
)

type Item interface {
	Get() *LogItem
	GetContent() []byte
	GetTime() *time.Time
}

type LogItem struct {
	File      string           // name of log file
	Time      *time.Time       // timestamp of a single line log
	Level     LevelType        // log level
	Host      string           // host ip
	Port      string           // port of component
	Component string           // name of component
	Line      []byte           // content of a single line log
	Type      ItemType         // type of log file
	Entries   map[string]Entry // field parsed from the log
}

type SlowLogItem struct {
	*LogItem
	content []byte
}

func (l *LogItem) Get() *LogItem {
	return l
}

func (l *LogItem) GetContent() []byte {
	return l.Line
}

func (l *LogItem) GetTime() *time.Time {
	return l.Time
}

func (l *SlowLogItem) Get() *LogItem {
	return l.LogItem
}

func (l *SlowLogItem) GetContent() []byte {
	return l.content
}
