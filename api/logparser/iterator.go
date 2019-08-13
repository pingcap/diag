package logparser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

const SlowLogQueryFileName = "tidb_slow_query.log"

type Iterator interface {
	Next() (Item, error)
	Close() error
}

type LogIterator struct {
	Filename  string
	Host      string
	Port      string
	Component string
	file      *os.File
	reader    *bufio.Reader
	begin     time.Time
	end       time.Time
	ParseFunc
	itemType ItemType
}

type SlowLogIterator struct {
	LogIterator
	time *time.Time
}

func (iter *LogIterator) NewLogItem(line []byte, t *time.Time, level LevelType) *LogItem {
	return &LogItem{
		File:      iter.Filename,
		Time:      t,
		Level:     level,
		Host:      iter.Host,
		Port:      iter.Port,
		Component: iter.Component,
		Line:      line,
		Type:      iter.itemType,
	}
}

// return value：
// first true：t is before the begin time
// second true：t is after the end time
func (iter *LogIterator) checkTime(t *time.Time) (bool, bool) {
	if t.Before(iter.begin) {
		return true, false
	}
	if t.After(iter.begin) && t.Before(iter.end) {
		return false, false
	}
	return false, true
}

func (iter *LogIterator) Close() error {
	if iter.file == nil {
		return fmt.Errorf("close a file which does not open")
	}
	return iter.file.Close()
}

func (iter *LogIterator) Next() (Item, error) {
	isPrefix := true
	var line, b []byte
	var err error
	for isPrefix {
		b, isPrefix, err = iter.reader.ReadLine()
		line = append(line, b...)
		if err != nil {
			return nil, err
		}
	}

	if iter.ParseFunc == nil {
		parseFunc := PeekLogFormat(line)
		if parseFunc == nil {
			log.Warnf("skip parse illegal log line: %s", line)
			return iter.Next()
		}
		iter.ParseFunc = parseFunc
	}
	ts, level, err := iter.ParseFunc(line)
	if err != nil {
		if err == io.EOF {
			return nil, err
		}
		parseFunc := PeekLogFormat(line)
		if parseFunc == nil {
			log.Warnf("skip parse illegal log line: %s", line)
			iter.ParseFunc = nil
			return iter.Next()
		}
		iter.ParseFunc = parseFunc
		ts, level, _ = iter.ParseFunc(line)
	}
	searching, stop := iter.checkTime(ts)
	if searching {
		return iter.Next()
	}
	if stop {
		return nil, io.EOF
	}
	item := iter.NewLogItem(line, ts, level)
	return item, nil
}

func (iter *SlowLogIterator) Next() (Item, error) {
	var line []byte
	var content []byte
	for {
		var b []byte
		var err error
		isPrefix := true
		line = []byte{}
		for isPrefix {
			b, isPrefix, err = iter.reader.ReadLine()
			line = append(line, b...)
			if err != nil {
				iter.time = nil
				return nil, err
			}
		}

		if !bytes.HasPrefix(line, []byte("#")) {
			break
		}

		t, err := iter.parseTime(line)
		if err != nil {
			iter.time = nil
			return nil, err
		}
		if t != nil {
			iter.time = t
			searching, stop := iter.checkTime(t)
			if searching {
				return iter.Next()
			}
			if stop {
				iter.time = nil
				return nil, io.EOF
			}
		}
		content = append(content, byte('\n'))
		content = append(content, line...)
	}
	logItem := iter.NewLogItem(line, iter.time, -1)

	item := &SlowLogItem{
		LogItem: logItem,
		content: content,
	}
	iter.time = nil
	return item, nil
}

func (iter *SlowLogIterator) parseTime(line []byte) (*time.Time, error) {
	if !SlowLogRE.Match(line) {
		return nil, nil
	}
	m := SlowLogRE.FindSubmatch(line)
	t, err := time.Parse(time.RFC3339Nano, string(m[1]))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func NewIterator(fw *FileWrapper, begin, end time.Time) (Iterator, error) {
	component, port, err := fw.parseFolderName()
	if err != nil {
		return nil, err
	}
	f, err := fw.open()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReaderSize(f, 64*1024)
	iter := LogIterator{
		Filename:  fw.Filename,
		Host:      fw.Host,
		Component: component,
		Port:      port,
		file:      f,
		reader:    reader,
		begin:     begin,
		end:       end,
	}
	switch component {
	case "tidb":
		if fw.Filename == SlowLogQueryFileName {
			iter.itemType = TypeTiDBSlowQuery
		} else {
			iter.itemType = TypeTiDB
		}
	case "tikv":
		iter.itemType = TypeTiKV
	case "pd":
		iter.itemType = TypePD
	default:
		return nil, io.EOF
	}
	var res Iterator
	if fw.Filename == SlowLogQueryFileName {
		res = &SlowLogIterator{iter, nil}
	} else {
		res = &iter
	}
	return res, nil
}

func PeekLogFormat(line []byte) ParseFunc {
	if _, _, err := parsePDLog(line); err == nil {
		return parsePDLog
	}

	if _, _, err := parseTiKVLog(line); err == nil {
		return parseTiKVLog
	}

	if _, _, err := parseUnifiedLog(line); err == nil {
		return parseUnifiedLog
	}

	return nil
}
