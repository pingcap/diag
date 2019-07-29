package main

import (
	"regexp"
	"time"
)

type TidbLogSpliter struct {
	LogSpliter
}

func NewTidbLogSpliter() Spliter {
	return &TidbLogSpliter{*NewLogSpliter(func(line string) *time.Time {
		re := regexp.MustCompile("^\\[([^\\]]+)\\]")
		if !re.MatchString(line) {
			return nil
		}
		m := re.FindStringSubmatch(line)
		t, e := time.Parse("2006/01/02 15:04:05.000 -07:00", m[1])
		if e != nil {
			return nil
		}
		return &t
	})}
}
