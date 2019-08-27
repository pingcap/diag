package iterator

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pingcap/tidb-foresight/log/item"
	"github.com/pingcap/tidb-foresight/log/parser"
)

// Only enable seek when position range is more than SEEK_THRESHOLD.
// The suggested value of SEEK_THRESHOLD is the biggest log size.
const SEEK_THRESHOLD = 1024 * 1024

// Define the slow query file name
const SlowLogQueryFileName = "tidb_slow_query.log"

// LogIterator implements Iterator and IteratorWithPeek interface.
// It's used for reading logs from log files one by one by their
// time.
type LogIterator struct {
	filename  string
	host      string
	port      string
	component string
	file      *os.File
	reader    *bufio.Reader
	begin     time.Time
	end       time.Time
	parsers   []parser.Parser
	current   item.Item
	nextError error
}

// Generate a new iterator from a specific file and a time range.
// The iterator should only return logs in the [begin, end] time
// range.
func New(fw *parser.FileWrapper, begin, end time.Time) (IteratorWithPeek, error) {
	component, port, err := fw.ParseFolderName()
	if err != nil {
		return nil, err
	}
	f, err := fw.Open()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReaderSize(f, 64*1024)
	iter := LogIterator{
		filename:  fw.Filename,
		host:      fw.Host,
		component: component,
		port:      port,
		file:      f,
		reader:    reader,
		begin:     begin,
		end:       end,
		parsers:   parser.List(),
	}

	if iter.itemType() == item.TypeInvalid {
		return nil, fmt.Errorf("invalid item type for component: %s", iter.component)
	}

	if err := iter.Seek(begin); err != nil {
		return nil, err
	} else {
		return &iter, nil
	}
}

// The Close method close all resources the iterator has.
func (iter *LogIterator) Close() error {
	if iter.file == nil {
		return fmt.Errorf("close a file which does not open")
	}
	return iter.file.Close()
}

// The Peek method returns next log, but don't change any state.
func (iter *LogIterator) Peek() item.Item {
	return iter.current
}

// The Next method return next log and error (if any), and read
// disk for future logs. If any erros happend in Next(), the error
// will not be returned immediately, instead, the error will be
// deplayed to the next call of Next().
func (iter *LogIterator) Next() (current item.Item, err error) {
	current = iter.current
	err = iter.nextError
	if err != nil {
		return
	}

	var line []byte
	if line, iter.nextError = iter.nextLine(); iter.nextError != nil {
		iter.current = nil
		return
	}

	item := iter.parse(line)
	if item != nil {
		if item.GetTime().After(iter.end) {
			iter.nextError = io.EOF
			iter.current = nil
		} else {
			iter.current = item
		}
		return
	}

	// all parsers has been tried, can't parse still,
	// we treat the line as the continuation of last log
	if current != nil {
		if iter.nextError = current.AppendContent(line); iter.nextError != nil {
			iter.current = nil
			return
		}
	}

	return iter.Next()
}

// The Seek method try to find the first log entity whose
// time is after the param point, if not found, io.EOF will
// be returned.
//
// params:
//     point: the start point of the logs
// return:
//		if found, nil
//		if not found, io.EOF
//		otherwise, other error
func (iter *LogIterator) Seek(point time.Time) error {
	info, err := iter.file.Stat()
	if err != nil {
		return err
	}

	return iter.seek(0, info.Size(), point)
}

// The implemention of Seek method, recursive call itself until found
// target log.
func (iter *LogIterator) seek(bpos, epos int64, point time.Time) error {
	if epos-bpos < SEEK_THRESHOLD {
		err := iter.carpet(bpos, epos, point)
		return err
	}

	begin := (bpos + epos) / 2
	end := epos
	if begin+SEEK_THRESHOLD < epos {
		end = begin + SEEK_THRESHOLD
	}

	err := iter.seek(begin, end, point)
	if err == io.EOF {
		return iter.seek(begin, epos, point)
	}
	if err == nil {
		// the may not be the very begining of target point
		if err := iter.seek(bpos, begin, point); err == nil {
			return nil
		} else if err == io.EOF {
			return iter.carpet(begin, end, point)
		} else {
			return err
		}
	}
	return err
}

// Carpet searching try to find the point and seek to it.
func (iter *LogIterator) carpet(bpos, epos int64, point time.Time) error {
	pos := bpos
	if _, err := iter.file.Seek(pos, 0); err != nil {
		return err
	}
	iter.reset()

	for pos < epos {
		line, err := iter.nextLine()
		if err != nil {
			return err
		}
		pos += int64(len(line) + 1)

		item := iter.parse(line)
		if item == nil {
			continue
		}
		if item.GetTime().After(point) {
			if item.GetTime().After(iter.end) {
				iter.current = nil
				iter.nextError = io.EOF
			} else {
				iter.current = item
			}
			return nil
		}
	}

	return io.EOF
}

// Reset iterater state, used after file seek.
func (iter *LogIterator) reset() {
	iter.reader.Reset(iter.file)
	iter.current = nil
	iter.nextError = nil
}

// Try to parse log item from a line, if the line is not start
// of a log, nil will be returned.
func (iter *LogIterator) parse(line []byte) item.Item {
	// try all parser
	for idx, parser := range iter.parsers {
		ts, level := parser.ParseHead(line)
		if ts != nil { // the line is the first line of the log
			// make the right parser the first one, so we can dirrectly hit next time
			iter.parsers[idx], iter.parsers[0] = iter.parsers[0], iter.parsers[idx]
			return iter.createLogItem(line, *ts, level)
		}
	}

	return nil
}

// Read a line from file.
func (iter *LogIterator) nextLine() ([]byte, error) {
	var line, b []byte
	var err error
	isPrefix := true
	for isPrefix {
		b, isPrefix, err = iter.reader.ReadLine()
		line = append(line, b...)
		if err != nil {
			return nil, err
		}
	}
	return line, nil
}

func (iter *LogIterator) createLogItem(line []byte, t time.Time, level item.LevelType) item.Item {
	return &item.LogItem{
		File:      iter.filename,
		Time:      t,
		Level:     level,
		Host:      iter.host,
		Port:      iter.port,
		Component: iter.component,
		Content:   line,
		Type:      iter.itemType(),
	}
}

func (iter *LogIterator) itemType() item.ItemType {
	switch iter.component {
	case "tidb":
		if iter.filename == SlowLogQueryFileName {
			return item.TypeTiDBSlowQuery
		} else {
			return item.TypeTiDB
		}
	case "tikv":
		return item.TypeTiKV
	case "pd":
		return item.TypePD
	default:
		return item.TypeInvalid
	}
}
