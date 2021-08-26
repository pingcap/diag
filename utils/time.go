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
	// try to parse input as a timestamp
	if t, err := strconv.ParseFloat(s, 64); err == nil {
		s, ns := math.Modf(t)
		ns = math.Round(ns*1000) / 1000
		return time.Unix(int64(s), int64(ns*float64(time.Second))).UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// try to parse input as some common time formats, all timestamps are supposed to
	// be localtime if not specified
	currTime := time.Now()
	for i, guess := range []string{
		"2006-01-02 15:04:05 -0700", // 0
		"2006-01-02 15:04 -0700",    // 1
		"2006-01-02 15 -0700",       // 2
		"2006-01-02 -0700",          // 3 ^-- full date time with timezone
		"2006-01-02 15:04:05",       // 4
		"2006-01-02 15:04",          // 5
		"2006-01-02 15",             // 6
		"2006-01-02",                // 7 ^-- full date time wo/ timezone
		"01-02 15:04:15 -0700",      // 8
		"01-02 15:04 -0700",         // 9
		"01-02 15 -0700",            // 10
		"01-02 -0700",               // 11 ^-- month date time with timezone
		"01-02 15:04:15",            // 12
		"01-02 15:04",               // 13
		"01-02 15",                  // 14
		"01-02",                     // 15 ^-- month date time wo/ timezone
		"15:04:15 -0700",            // 16
		"15:04 -0700",               // 17
		"15 -0700",                  // 18 ^-- hour time with timezone
		"15:04:15",                  // 19
		"15:04",                     // 20
		"15",                        // 21 ^-- hour time wo/ timezone
	} {
		if t, err := time.Parse(guess, s); err == nil {
			parsedLoc := t.Location()
			if i > 7 && t.Year() == 0 {
				t = time.Date(
					currTime.Year(), t.Month(),
					t.Day(), t.Hour(),
					t.Minute(), t.Second(),
					t.Nanosecond(), currTime.Location(),
				)
			}
			if i > 15 && t.Month() == 1 {
				t = time.Date(
					currTime.Year(), currTime.Month(),
					t.Day(), t.Hour(),
					t.Minute(), t.Second(),
					t.Nanosecond(), currTime.Location(),
				)
			}
			if i > 15 && t.Day() == 1 {
				t = time.Date(
					currTime.Year(), currTime.Month(),
					currTime.Day(), t.Hour(),
					t.Minute(), t.Second(),
					t.Nanosecond(), currTime.Location(),
				)
			}
			// set timezone if specified in input
			if i <= 3 || (i >= 8 && i <= 11) || (i >= 16 && i <= 18) {
				t = time.Date(
					t.Year(), t.Month(),
					t.Day(), t.Hour(),
					t.Minute(), t.Second(),
					t.Nanosecond(), parsedLoc,
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
