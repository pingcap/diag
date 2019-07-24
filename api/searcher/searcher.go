package searcher

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Searcher struct {
	m map[string]LogIter
	l sync.Mutex
}

//type Log struct {
//	Ip        string    `json:"ip"`
//	Port      string    `json:"port"`
//	File      string    `json:"file"`
//	Time      time.Time `json:"time"`
//	Component string    `json:"component"`
//	Level     string    `json:"level"`
//	Content   string    `json:"content"`
//}

type LogIter interface {
	Next() (*Item, error)
	Close() error
}

type IterWithAccessTime struct {
	iter   LogIter
	access time.Time
	l      sync.Mutex
}

func NewIter(iter LogIter) *IterWithAccessTime {
	return &IterWithAccessTime{
		iter:   iter,
		access: time.Now(),
	}
}

func (i *IterWithAccessTime) Next() (*Item, error) {
	i.l.Lock()
	defer i.l.Unlock()
	i.access = time.Now()
	if i.iter != nil {
		return i.iter.Next()
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

func (s *Searcher) Search(dir string, text string, token string) (LogIter, string, error) {
	if token == "" {
		token = uuid.New().String()
		i, err := SearchLog(dir, text)
		if err != nil {
			return nil, token, err
		}
		iter := NewIter(i)
		go s.Gc(token, iter)
		return iter, token, err
	} else {
		return s.GetIter(token), token, nil
	}
}


//func GetIterFromToken(token string) LogIter {
//	return &Mock{}
//}
