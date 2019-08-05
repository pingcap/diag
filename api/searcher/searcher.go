package searcher

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Searcher struct {
	m map[string]LogIter
	l sync.Mutex
}

type LogIter interface {
	Next() (*Item, error)
	Close() error
}

type IterWithAccessTime struct {
	iter   LogIter
	access time.Time
	begin  time.Time
	end    time.Time
	level  string
	l      sync.Mutex
}

func NewIter(iter LogIter, begin, end time.Time, level string) *IterWithAccessTime {
	return &IterWithAccessTime{
		iter:   iter,
		access: time.Now(),
		begin:  begin,
		end:    end,
		level:  level,
	}
}

func (i *IterWithAccessTime) Next() (*Item, error) {
	i.l.Lock()
	defer i.l.Unlock()
	i.access = time.Now()
	levelMap := map[string]LevelType{
		"OTHERES": -1,
		"INFO":    LevelINFO,
		"DEBUG":   LevelDEBUG,
		"WARNING": LevelWARN,
		"ERROR":   LevelERROR,
	}

	if i.iter != nil {
		for {
			item, err := i.iter.Next()
			if item == nil || err != nil {
				return nil, err
			}
			// check if time in range
			if item.Time == nil { // FIXME: item.Time should NOT be nil
				log.Warn("get nil time while search log")
				continue
			}
			log.Info(*item.Time, i.begin, i.end)
			if item.Time.Before(i.begin) {
				continue
			}
			if item.Time.After(i.end) {
				return nil, nil
			}
			// check if level is correct
			if i.level != "" && item.Level != levelMap[i.level] {
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

func NewSearcher() *Searcher {
	return &Searcher{
		m: make(map[string]LogIter),
	}
}

func (s *Searcher) SetIter(token string, iter LogIter) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[token] = iter
}

func (s *Searcher) GetIter(token string) LogIter {
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

func (s *Searcher) Search(dir string, begin, end time.Time, level, text, token string) (LogIter, string, error) {
	if token == "" {
		token = uuid.New().String()
		i, err := SearchLog(dir, text)
		if err != nil {
			return nil, token, err
		}
		iter := NewIter(i, begin, end, level)
		go s.Gc(token, iter)
		return iter, token, err
	} else {
		return s.GetIter(token), token, nil
	}
}
