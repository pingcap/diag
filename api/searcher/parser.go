package searcher

import (
	"bufio"
	"bytes"
	"errors"
	"math"
	"os"
	"path"
	"time"
)

type Entry struct {
	Type  string // golang type of Value
	Value interface{}
}

type Item struct {
	File      string           // name of log file
	Time      *time.Time       // timestamp of a single line log
	Level     LevelType        // log level
	Host      string           // host ip
	Port      string           // port of component
	Component string           // name of component
	Line      string           // content of a single line log
	Type      ItemType         // type of log file
	Entries   map[string]Entry // field parsed from the log
}

type Parser interface {
	Next() error  // parse next log, assign it to currentLog
	Close() error // close file descriptor
	GetTs() int64 // get timestamp of currentLog
	GetCurrentLog() *Item
}

type BaseParser struct {
	Filename    string
	Host        string
	Port        string
	Component   string
	File        *os.File
	Reader      *bufio.Reader
	SearchBytes []byte
	current     *Item
}

func (p *BaseParser) NewLogItem(line string, ts *time.Time, level LevelType) *Item {
	return &Item{
		File:      p.Filename,
		Time:      ts,
		Level:     level,
		Host:      p.Host,
		Port:      p.Port,
		Component: p.Component,
		Line:      line,
	}
}

func (p *BaseParser) Next() (string, error) {
	var line []byte
	var err error
	for {
		// Note: Only the first 4094 byte of this row can be read out
		line, _, err = p.Reader.ReadLine()
		if err != nil {
			return "", err
		}
		if bytes.Contains(line, p.SearchBytes) {
			break
		}
	}
	return string(line), nil
}

func (p *BaseParser) GetTs() int64 {
	if p.current.Time == nil {
		return math.MaxInt64
	}
	return p.current.Time.Unix()
}

func (p *BaseParser) GetCurrentLog() *Item {
	return p.current
}

func (p *BaseParser) Close() error {
	if p.File == nil {
		return errors.New("close a file which does not open")
	}
	return p.File.Close()
}

// NewParser return a Parser for specific log type
func NewParser(cluster, host, folder, filename, searchStr string) (Parser, error) {
	component, port, err := parseFolder(folder)
	if err != nil {
		return nil, err
	}
	filePath := path.Join(cluster, host, folder, filename)
	f, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(f)
	base := BaseParser{
		Filename:    filename,
		Host:        host,
		Component:   component,
		Port:        port,
		File:        f,
		Reader:      reader,
		SearchBytes: []byte(searchStr),
	}
	var parser Parser
	switch component {
	case "tidb":
		if filename == "tidb_slow_query.log" {
			parser = &TidbSlogQueryParser{base}
		} else {
			parser = &TidbLogParser{base}
		}
	case "tikv":
		parser = &TikvLogParser{base}
	case "pd":
		parser = &PDLogParser{base}
	default:
		return nil, errors.New("newParser: unsupported component type")
	}
	err = parser.Next()
	if err != nil {
		return nil, err
	}
	return parser, nil
}
