package searcher

import (
	"io"
	"time"

	"github.com/pingcap/tidb-foresight/api/logparser"
	log "github.com/sirupsen/logrus"
)

type IterWrapper struct {
	iter logparser.Iterator
	item logparser.Item
}

type Sequence struct {
	slice []IterWrapper
}

// choose the log with earlier timestamp
func (s *Sequence) Next() (logparser.Item, error) {
	var res logparser.Item
	currentTs := time.Now()
	for i := 0; i < len(s.slice); i++ {
		w := s.slice[i]
		if w.item == nil {
			item, err := w.iter.Next()
			if err != nil {
				if err == io.EOF {
					s.Remove(i)
					i--
					continue
				}
				return nil, err
			}
			w.item = item
		}
		ts := w.item.GetTime()
		if ts.Before(currentTs) {
			currentTs = *ts
			res = w.item
		}
	}
	return res, nil
}

func (s *Sequence) Close() error {
	for _, item := range s.slice {
		err := item.iter.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Sequence) Add(iter logparser.Iterator) {
	if iter == nil {
		return
	}
	if len(s.slice) == 0 {
		s.slice = []IterWrapper{{
			iter: iter,
			item: nil,
		}}
	} else {
		s.slice = append(s.slice, IterWrapper{
			iter: iter,
			item: nil,
		})
	}
}

func (s *Sequence) Remove(index int) {
	if index < 0 || index > len(s.slice) {
		return
	}
	s.slice[index].iter.Close()
	s.slice = append(s.slice[:index], s.slice[index+1:]...)
}

// SearchLog open all log files in the directory,
// analyze each log in each file by merge sort (from old to new by timestamp),
// return the constructed LogIter object and provide the Next function for external call.
func SearchLog(src string, begin, end time.Time) (*Sequence, error) {
	sequence := &Sequence{}
	files, err := logparser.ResolveDir(src)
	if err != nil {
		return nil, err
	}
	for _, fw := range files {
		iter, err := logparser.NewIterator(fw, begin, end)
		if err != nil {
			if err != io.EOF {
				log.Warnf("create log iterator err: %s", err)
			}
			continue
		}
		sequence.Add(iter)
	}
	return sequence, nil
}
