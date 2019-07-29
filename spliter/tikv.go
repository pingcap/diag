package main

import (
	"time"
)

type TikvLogSpliter struct {
	LogSpliter
}

func NewTikvLogSpliter() Spliter {
	return &TikvLogSpliter{*NewLogSpliter(func(line string) *time.Time {
		if len(line) < 23 {
			return nil
		}
		loc, _ := time.LoadLocation("Asia/Shanghai")
		t, e := time.ParseInLocation("2006/01/02 15:04:05.000", line[0:23], loc)
		if e != nil {
			return nil
		}
		return &t
	})}
}
