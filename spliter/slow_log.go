package main

import (
	"io"
	"time"
	"bufio"
	"regexp"
)

const (
	SEARCHING = iota
	WRITING
	SKIPING
)

type SlowLogSpliter struct {
	status int
}

func NewSlowLogSpliter() Spliter {
	return &SlowLogSpliter{SEARCHING}
}

func (s *SlowLogSpliter) Spit(r io.Reader, w io.Writer, begin, end time.Time) error {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		switch s.status {
		case SEARCHING:
			// find begin of a log，aka # Time
			logT := s.parseTime(line)

			// continue if no time present
			if logT == nil {
				continue
			}

			if logT.After(end) {
				return nil
			}
			// if time match, write the log it belongs to, otherwise, drop the log it belongs to.
			if logT.After(begin) {
				s.status = WRITING
				_, err := w.Write([]byte(line + "\n"))
				if err != nil {
					return err
				}
			} else {
				s.status = SKIPING
			}
		case WRITING:
			_, err := w.Write([]byte(line + "\n"))
			if err != nil {
				return err
			}
		}

		if s.isQuery(line) {
			s.status = SEARCHING
		}
    }

	return nil
}

func (s *SlowLogSpliter) parseTime(line string) *time.Time {
	re := regexp.MustCompile("^# Time: (.*)$")
	if !re.MatchString(line) {
		return nil
	}
	m := re.FindStringSubmatch(line)
	t, e := time.Parse(time.RFC3339, m[1])
	if e != nil {
		return nil
	}
	return &t
}

func (s *SlowLogSpliter) isQuery(line string) bool {
	return len(line) > 0 && line[0] != '#'
}