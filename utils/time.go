package utils

import (
	"math"
	"strconv"
	"time"

	"github.com/pingcap/errors"
)

var (
	minTime = time.Unix(math.MinInt64/1000+62135596801, 0).UTC()
	maxTime = time.Unix(math.MaxInt64/1000-62135596801, 999999999).UTC()

	minTimeFormatted = minTime.Format(time.RFC3339Nano)
	maxTimeFormatted = maxTime.Format(time.RFC3339Nano)
)

// ParseTime converts a string to time.Time, ported from Prometheus: web/api/v1/api.go
func ParseTime(s string) (time.Time, error) {
	if t, err := strconv.ParseFloat(s, 64); err == nil {
		s, ns := math.Modf(t)
		ns = math.Round(ns*1000) / 1000
		return time.Unix(int64(s), int64(ns*float64(time.Second))).UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}

	// try to parse some common time formats, all timezones are supposed to be localtime
	currTime := time.Now()
	for i, guess := range []string{
		"2006-01-02 04:05:07",
		"2006-01-02 04:05",
		"2006-01-02 04",
		"2006-01-02",
		"01-02 04:05:07",
		"01-02 04:05",
		"01-02 04",
		"01-02",
		"04:05:07",
		"04:05",
		"04",
	} {
		if t, err := time.Parse(guess, s); err == nil {
			if i > 3 && t.Year() == 0 {
				t = time.Date(
					currTime.Year(), t.Month(),
					t.Day(), t.Hour(),
					t.Minute(), t.Second(),
					t.Nanosecond(), currTime.Location(),
				)
			}
			if i > 7 && t.Month() == 1 {
				t = time.Date(
					currTime.Year(), currTime.Month(),
					t.Day(), t.Hour(),
					t.Minute(), t.Second(),
					t.Nanosecond(), currTime.Location(),
				)
			}
			if i > 7 && t.Day() == 1 {
				t = time.Date(
					currTime.Year(), currTime.Month(),
					currTime.Day(), t.Hour(),
					t.Minute(), t.Second(),
					t.Nanosecond(), currTime.Location(),
				)
			}
			return t, nil
		}
	}

	// Stdlib's time parser can only handle 4 digit years. As a workaround until
	// that is fixed we want to at least support our own boundary times.
	// Context: https://github.com/prometheus/client_golang/issues/614
	// Upstream issue: https://github.com/golang/go/issues/20555
	switch s {
	case minTimeFormatted:
		return minTime, nil
	case maxTimeFormatted:
		return maxTime, nil
	}
	return time.Time{}, errors.Errorf("cannot parse %q to a valid timestamp", s)
}
