package item

import (
	"fmt"
	"time"
)

const MAX_LOG_SIZE = 1024 * 1024

// The LogItem struct implements Item interface
type LogItem struct {
	File      string    // name of log file
	Time      time.Time // timestamp of a single line log
	Level     LevelType // log level
	Host      string    // host ip
	Port      string    // port of component
	Component string    // name of component
	Content   []byte    // content of a entire line log
	Type      ItemType  // type of log file
}

func (l *LogItem) GetHost() string {
	return l.Host
}

func (l *LogItem) GetPort() string {
	return l.Port
}

func (l *LogItem) GetComponent() string {
	return l.Component
}

func (l *LogItem) GetFileName() string {
	return l.File
}

func (l *LogItem) GetTime() time.Time {
	return l.Time
}

func (l *LogItem) GetLevel() LevelType {
	return l.Level
}

func (l *LogItem) GetContent() []byte {
	return l.Content
}

func (l *LogItem) AppendContent(content []byte) error {
	if len(l.Content) > MAX_LOG_SIZE {
		return fmt.Errorf("log size exceeds limit, log content:\n%s\n", string(l.Content))
	}
	l.Content = append(l.Content, byte('\n'))
	l.Content = append(l.Content, content...)
	return nil
}
