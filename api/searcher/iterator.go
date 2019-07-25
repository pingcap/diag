package searcher

import (
	"errors"
	"io"
	"io/ioutil"
	"math"
	"path"
	"strings"
)

type Iterator struct {
	parsers []Parser
}

// NewIterator open all files in the root directory,
// generate different types of log file parser and return the Iterator.
// when call the Next function of Iterator,
// it will return a single line log Item object parsed from all the log file.
func NewIterator(cluster string, searchStr string) (*Iterator, error) {
	logIterator := &Iterator{}
	dir, err := ioutil.ReadDir(cluster)
	if err != nil {
		return nil, err
	}
	for _, fi := range dir {
		host := fi.Name() // {host_ip}
		if !fi.IsDir() {
			continue
		}
		dirPath := path.Join(cluster, host)
		dir, err := ioutil.ReadDir(dirPath)
		if err != nil {
			return nil, err
		}
		for _, fi := range dir {
			folder := fi.Name() // {component_name}-{port}
			if !fi.IsDir() {
				continue
			}
			dirPath := path.Join(dirPath, folder)
			dir, err := ioutil.ReadDir(dirPath)
			if err != nil {
				return nil, err
			}
			for _, fi := range dir {
				filename := fi.Name()
				if fi.IsDir() {
					continue
				}
				parser, err := NewParser(cluster, host, folder, filename, searchStr)
				if err != nil {
					if err == io.EOF {
						continue
					}
					return nil, err
				}
				if parser != nil {
					logIterator.AddParser(parser)
				}
			}
		}
	}
	return logIterator, nil
}

func parseFolder(name string) (string, string, error) {
	s := strings.Split(name, "-")
	if len(s) < 2 {
		return "", "", errors.New("wrong folder name: " + name)
	}
	return s[0], s[1], nil
}

func (l *Iterator) Next() (*Item, error) {
	if len(l.parsers) == 0 {
		return nil, nil
	}
	var res *Item
	currentTs := int64(math.MaxInt64)
	for _, parser := range l.parsers {
		// choose the log with earlier timestamp
		log := parser.GetCurrentLog()
		if log == nil {
			continue
		}
		ts := parser.GetTs()
		if ts < currentTs {
			res = log
			currentTs = ts
			err := parser.Next()
			if err != nil {
				if err == io.EOF {
					continue
				}
				return nil, err
			}
		}
	}
	return res, nil
}

func (l *Iterator) Close() error {
	for _, parser := range l.parsers {
		err := parser.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Iterator) AddParser(parser Parser) {
	if l.parsers == nil {
		l.parsers = []Parser{parser}
	} else {
		l.parsers = append(l.parsers, parser)
	}
}

// SearchLog open all log files in the directory,
// analyze each log in each file by merge sort (from old to new by timestamp),
// return the constructed LogIter object and provide the Next function for external call.
func SearchLog(dir string, text string) (*Iterator, error) {
	return NewIterator(dir, text)
}
