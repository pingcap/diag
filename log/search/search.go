package search

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pingcap/tidb-foresight/log/item"
	"github.com/pingcap/tidb-foresight/log/iterator"
	"github.com/pingcap/tidb-foresight/log/parser"
)

type Searcher interface {
	Search(dir string, begin, end time.Time, level, text, token string) (iterator.Iterator, string, error)
}

type searcher struct {
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

func (i *IterWithAccessTime) Next() (item.Item, error) {
	i.l.Lock()
	defer i.l.Unlock()
	i.access = time.Now()

	if i.iter != nil {
		for {
			item, err := i.iter.Next()
			if err != nil {
				return nil, err
			}
			if !bytes.Contains(item.GetContent(), i.search) {
				continue
			}
			if i.level != "" && item.GetLevel() != parser.ParseLogLevel([]byte(i.level)) {
				continue
			}
			return item, nil
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

func NewSearcher() Searcher {
	return &searcher{
		m: make(map[string]*IterWithAccessTime),
	}
}

func (s *searcher) SetIter(token string, iter *IterWithAccessTime) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[token] = iter
}

func (s *searcher) GetIter(token string) *IterWithAccessTime {
	s.l.Lock()
	defer s.l.Unlock()
	return s.m[token]
}

func (s *searcher) DelIter(token string) {
	s.l.Lock()
	defer s.l.Unlock()
	delete(s.m, token)
}

func (s *searcher) Gc(token string, iter *IterWithAccessTime) {
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

func (s *searcher) Search(dir string, begin, end time.Time, level, text, token string) (iterator.Iterator, string, error) {
	if token == "" {
		token = uuid.New().String()
		i, err := NewSequence(dir, begin, end)
		if err != nil {
			return nil, token, err
		}
		iter := NewIter(i, text, level)
		go s.Gc(token, iter)
		return iter, token, err
	} else {
		if iter := s.GetIter(token); iter == nil {
			return nil, token, fmt.Errorf("not found")
		} else {
			return iter, token, nil
		}
	}
}
