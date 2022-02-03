package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type ReadableSize = uint64

const UNIT uint64 = 1

// for readable byte size

const BinaryDataMagnitude uint64 = 1024
const B uint64 = UNIT
const KIB uint64 = UNIT * BinaryDataMagnitude
const MIB uint64 = KIB * BinaryDataMagnitude
const GIB uint64 = MIB * BinaryDataMagnitude
const TIB uint64 = GIB * BinaryDataMagnitude
const PIB uint64 = TIB * BinaryDataMagnitude

func ParseReadableSize(s string) (ReadableSize, error) {
	sizeStr := strings.TrimSpace(s)
	if len(sizeStr) == 0 {
		return 0, fmt.Errorf("%q is not a vaild size", s)
	}
	idx := strings.LastIndexFunc(sizeStr, func(r rune) bool {
		if unicode.IsDigit(r) {
			return true
		}
		switch r {
		case '.', 'e', 'E', '-', '+':
			return true
		default:
			return false
		}
	})
	if idx < 0 {
		return 0, fmt.Errorf("%q is not a vaild size", s)
	}
	unitStr := sizeStr[idx+1:]
	var unit uint64
	switch unitStr {
	case "K", "KB", "KiB":
		unit = KIB
	case "M", "MB", "MiB":
		unit = MIB
	case "G", "GB", "GiB":
		unit = GIB
	case "T", "TB", "TiB":
		unit = TIB
	case "P", "PB", "PiB":
		unit = PIB
	case "", "B":
		unit = B
	default:
		return 0, fmt.Errorf("%q is not a supported unit", unitStr)
	}
	sizeVal, err := strconv.ParseFloat(sizeStr[0:idx+1], 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a vaild size", s)
	}
	return ReadableSize(sizeVal * float64(unit)), nil
}

// MustCmpReadableSize return 0, -1, 1; 0 left == right, 1 left > right, -1 left < right
// Only use this method in gengine
func MustCmpReadableSize(left string, right string) int {
	leftSize, err := ParseReadableSize(left)
	if err != nil {
		panic(err)
	}
	rightSize, err := ParseReadableSize(right)
	if err != nil {
		panic(err)
	}
	if leftSize == rightSize {
		return 0
	}
	if leftSize > rightSize {
		return 1
	}
	return -1
}

// MustCmpDuration return 0, -1, 1; 0 left == right, 1 left > right, -1 left < right
// Only use this method in gengine
func MustCmpDuration(left string, right string) int {
	leftDur, err := ParseReadableDuration(left)
	if err != nil {
		panic(err)
	}
	rightDur, err := ParseReadableDuration(right)
	if err != nil {
		panic(err)
	}
	if leftDur == rightDur {
		return 0
	}
	if leftDur > rightDur {
		return 1
	}
	return -1
}

var durationRE = regexp.MustCompile("^(([0-9]+)d)?(([0-9]+)h)?(([0-9]+)m)?(([0-9]+)s)?(([0-9]+)ms)?(([0-9]+)us)?$")

// ParseReadableDuration is copied from prometheus ParseDuration function.
func ParseReadableDuration(s string) (time.Duration, error) {
	switch s {
	case "0":
		// Allow 0 without a unit.
		return 0, nil
	case "":
		return 0, fmt.Errorf("%q is not a vaild duration", s)
	}
	matches := durationRE.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("%q is not a vaild duration", s)
	}
	var dur time.Duration

	// Parse the match at pos `pos` in the regex and use `mult` to turn that
	// into ms, then add that value to the total parsed duration.
	var overflowErr error
	m := func(pos int, mult time.Duration) {
		if matches[pos] == "" {
			return
		}
		n, _ := strconv.Atoi(matches[pos])

		// Check if the provided duration overflows time.Duration (> ~ 290years).
		if n > int((1<<63-1)/mult/time.Millisecond) {
			overflowErr = errors.New("duration out of range")
		}
		d := time.Duration(n) * time.Microsecond
		dur += d * mult

		if dur < 0 {
			overflowErr = errors.New("duration out of range")
		}
	}

	m(2, 10e6*60*60*24) // d
	m(4, 10e6*60*60)    // h
	m(6, 10e6*60)       // m
	m(8, 10e6)          // s
	m(10, 10e3)         // ms
	m(12, 1)            // us

	return dur, overflowErr
}
