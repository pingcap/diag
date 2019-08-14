package searcher

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pingcap/tidb-foresight/log/parser"
)

type Searcher struct {
	m map[string]*IterWithAccessTime
	l sync.Mutex
}

type IterWithAccessTime struct {
	iter   *Sequence
	access time.Time
	search []byte
	level  string
	l      sync.Mutex
}

func NewIter(iter *Sequence, search, level string) *IterWithAccessTime {
	return &IterWithAccessTime{
		iter:   iter,
		access: time.Now(),
		search: []byte(search),
		level:  level,
	}
}

var levelMap = map[string]parser.LevelType{
	"OTHERES": -1,
	"INFO":    parser.LevelINFO,
	"DEBUG":   parser.LevelDEBUG,
	"WARNING": parser.LevelWARN,
	"ERROR":   parser.LevelERROR,
}

func (i *IterWithAccessTime) Next() (*parser.LogItem, error) {
	i.l.Lock()
	defer i.l.Unlock()
	i.access = time.Now()

	if i.iter != nil {
		for {
			item, err := i.iter.Next()
			if err != nil {
				return nil, err
			}
			logItem := item.Get()
			if !bytes.Contains(logItem.Line, i.search) {
				continue
			}
			if i.level != "" && logItem.Level != levelMap[i.level] {
				continue
			}
			return logItem, nil
		}
	} else {
		return nil, errors.New("log file closed")
	}
}

func (i *IterWithAccessTime) Close() error {
	i.l.Lock()
	defer i.l.Unlock()

	// in case of multiple close
	if i.iter == nil {
		return nil
	}
	iter := i.iter
	i.iter = nil

	return iter.Close()
}

func (i *IterWithAccessTime) GetAccessTime() time.Time {
	i.l.Lock()
	defer i.l.Unlock()
	return i.access
}

func NewSearcher() *Searcher {
	return &Searcher{
		m: make(map[string]*IterWithAccessTime),
	}
}

func (s *Searcher) SetIter(token string, iter *IterWithAccessTime) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[token] = iter
}

func (s *Searcher) GetIter(token string) *IterWithAccessTime {
	s.l.Lock()
	defer s.l.Unlock()
	return s.m[token]
}

func (s *Searcher) DelIter(token string) {
	s.l.Lock()
	defer s.l.Unlock()
	delete(s.m, token)
}

func (s *Searcher) Gc(token string, iter *IterWithAccessTime) {
	const DURATION = 60 * time.Second

	s.SetIter(token, iter)

	for {
		time.Sleep(DURATION - time.Since(iter.GetAccessTime()))

		if iter.GetAccessTime().Add(DURATION).Before(time.Now()) {
			s.DelIter(token)
			iter.Close()
			break
		}
	}
}

func (s *Searcher) Search(dir string, begin, end time.Time, level, text, token string) (*IterWithAccessTime, string, error) {
	if token == "" {
		token = uuid.New().String()
		i, err := SearchLog(dir, begin, end)
		if err != nil {
			return nil, token, err
		}
		iter := NewIter(i, text, level)
		go s.Gc(token, iter)
		return iter, token, err
	} else {
		return s.GetIter(token), token, nil
	}
}
