package search

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/pingcap/diag/collector/log/item"
	"github.com/pingcap/diag/collector/log/iterator"
	"github.com/pingcap/diag/collector/log/parser"
	log "github.com/sirupsen/logrus"
)

type Sequence struct {
	slice []iterator.IteratorWithPeek
}

// choose the log with earlier timestamp
func (s *Sequence) Next() (item.Item, error) {
	var currentIter iterator.IteratorWithPeek
	currentTs := time.Now()

	for i := 0; i < len(s.slice); i++ {
		iter := s.slice[i]
		if iter.Peek() == nil {
			if _, err := iter.Next(); err != nil {
				if err == io.EOF {
					s.Remove(i)
					i--
					continue
				}
				return nil, err
			}
			return nil, fmt.Errorf("log iterator return empty item without error")
		}
		ts := iter.Peek().GetTime()
		if ts.Before(currentTs) {
			currentTs = ts
			currentIter = iter
		}
	}

	if currentIter == nil {
		return nil, io.EOF
	}
	return currentIter.Next()
}

func (s *Sequence) Close() error {
	for _, iter := range s.slice {
		err := iter.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Sequence) Add(iter iterator.IteratorWithPeek) {
	if iter == nil {
		return
	}
	if len(s.slice) == 0 {
		s.slice = []iterator.IteratorWithPeek{iter}
	} else {
		s.slice = append(s.slice, iter)
	}
}

func (s *Sequence) Remove(index int) {
	if index < 0 || index > len(s.slice) {
		return
	}
	s.slice[index].Close()
	s.slice = append(s.slice[:index], s.slice[index+1:]...)
}

// SearchLog open all log files in the directory,
// analyze each log in each file by merge sort (from old to new by timestamp),
// return the constructed LogIter object and provide the Next function for external call.
func NewSequence(src string, begin, end time.Time) (*Sequence, error) {
	sequence := &Sequence{}
	files, err := parser.ResolveDir(src)
	if err != nil {
		return nil, err
	}

	iters := make(chan iterator.IteratorWithPeek, len(files))
	wg := sync.WaitGroup{}
	wg.Add(len(files))

	for _, fw := range files {
		go func(fw *parser.FileWrapper) {
			if iter, err := iterator.New(fw, begin, end); err != nil {
				if err != io.EOF {
					log.Warnf("create log iterator err: %s", err)
				}
			} else {
				iters <- iter
			}
			wg.Done()
		}(fw)
	}
	wg.Wait()
	close(iters)

	for iter := range iters {
		sequence.Add(iter)
	}
	return sequence, nil
}
