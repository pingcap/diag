package main

import (
	"io"
	"os"
	"fmt"
	"path"
	"time"
	"bufio"
	"strings"
	"io/ioutil"
)


type Spliter interface {
	Spit(io.Reader, io.Writer, time.Time, time.Time) error
}

func NewSpliter(logName string) Spliter {
	if strings.Contains(logName, "tidb_slow_query.log") {
		return NewSlowLogSpliter()
	} else if strings.Contains(logName, "tidb.log") {
		return NewTidbLogSpliter()
	} else if strings.Contains(logName, "tikv.log") {
		return NewTikvLogSpliter()
	} else if strings.Contains(logName, "pd.log") {
		return NewPdLogSpliter()
	} else {
		return nil
	}
}

func copy(src, dst string, begin, end time.Time) error {
	f, err := os.Stat(src)
	if err != nil {
		fmt.Println("stat:", err)
		return err
	}
	switch mode := f.Mode(); {
	case mode.IsDir():
		if err = os.MkdirAll(dst, os.ModePerm); err != nil {
			fmt.Println("mkdir:", err)
			return err
		}
		files, err := ioutil.ReadDir(src)
		if err != nil {
			fmt.Println("readdir:", err)
			return err
		}
		for _, f := range files {
			if err = copy(path.Join(src, f.Name()), path.Join(dst, f.Name()), begin, end); err != nil {
				return err
			}
		}
	case mode.IsRegular():
		spliter := NewSpliter(path.Base(src))
		if spliter == nil {
			return nil
		}
		from, err := os.Open(src)
		if err != nil {
			fmt.Println("open:", err)
			return err
		}
		defer from.Close()
		to, err := os.Create(dst)
		if err != nil {
			fmt.Println("create:", err)
			return err
		}
		defer to.Close()
		return spliter.Spit(from, to, begin, end)
	}
	return nil
}

type LogSpliter struct {
	matched bool
	timeParser func(string) *time.Time
}

func NewLogSpliter(timeParser func(string) *time.Time) *LogSpliter {
	return &LogSpliter{matched: false, timeParser: timeParser}
}

func (s *LogSpliter) Spit(r io.Reader, w io.Writer, begin, end time.Time) error {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		t := s.timeParser(line)
		if t != nil {
			if t.After(end) {
				// no more matched
				return nil
			}
			if t.After(begin) {
				s.matched = true
				w.Write([]byte(line + "\n"))
			}
		} else {
			if s.matched {
				// it belongs to the last log
				w.Write([]byte(line + "\n"))
			}
		}
    }

	return nil
}