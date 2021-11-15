// Copyright 2016 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"fmt"
	gotime "time"

	"github.com/pingcap/errors"
)

// CoreTime is the internal struct type for Time.
type CoreTime uint64

// ZeroCoreTime is the zero value for TimeInternal type.
var ZeroCoreTime = CoreTime(0)

// String implements fmt.Stringer.
func (t CoreTime) String() string {
	return fmt.Sprintf("{%d %d %d %d %d %d %d}", t.getYear(), t.getMonth(), t.getDay(), t.getHour(), t.getMinute(), t.getSecond(), t.getMicrosecond())
}

func (t CoreTime) getYear() uint16 {
	return uint16((uint64(t) & yearBitFieldMask) >> yearBitFieldOffset)
}

func (t *CoreTime) setYear(year uint16) {
	*(*uint64)(t) &= ^yearBitFieldMask
	*(*uint64)(t) |= (uint64(year) << yearBitFieldOffset) & yearBitFieldMask
}

// Year returns the year value.
func (t CoreTime) Year() int {
	return int(t.getYear())
}

func (t CoreTime) getMonth() uint8 {
	return uint8((uint64(t) & monthBitFieldMask) >> monthBitFieldOffset)
}

func (t *CoreTime) setMonth(month uint8) {
	*(*uint64)(t) &= ^monthBitFieldMask
	*(*uint64)(t) |= (uint64(month) << monthBitFieldOffset) & monthBitFieldMask
}

// Month returns the month value.
func (t CoreTime) Month() int {
	return int(t.getMonth())
}

func (t CoreTime) getDay() uint8 {
	return uint8((uint64(t) & dayBitFieldMask) >> dayBitFieldOffset)
}

func (t *CoreTime) setDay(day uint8) {
	*(*uint64)(t) &= ^dayBitFieldMask
	*(*uint64)(t) |= (uint64(day) << dayBitFieldOffset) & dayBitFieldMask
}

// Day returns the day value.
func (t CoreTime) Day() int {
	return int(t.getDay())
}

func (t CoreTime) getHour() uint8 {
	return uint8((uint64(t) & hourBitFieldMask) >> hourBitFieldOffset)
}

func (t *CoreTime) setHour(hour uint8) {
	*(*uint64)(t) &= ^hourBitFieldMask
	*(*uint64)(t) |= (uint64(hour) << hourBitFieldOffset) & hourBitFieldMask
}

// Hour returns the hour value.
func (t CoreTime) Hour() int {
	return int(t.getHour())
}

func (t CoreTime) getMinute() uint8 {
	return uint8((uint64(t) & minuteBitFieldMask) >> minuteBitFieldOffset)
}

func (t *CoreTime) setMinute(minute uint8) {
	*(*uint64)(t) &= ^minuteBitFieldMask
	*(*uint64)(t) |= (uint64(minute) << minuteBitFieldOffset) & minuteBitFieldMask
}

// Minute returns the minute value.
func (t CoreTime) Minute() int {
	return int(t.getMinute())
}

func (t CoreTime) getSecond() uint8 {
	return uint8((uint64(t) & secondBitFieldMask) >> secondBitFieldOffset)
}

func (t *CoreTime) setSecond(second uint8) {
	*(*uint64)(t) &= ^secondBitFieldMask
	*(*uint64)(t) |= (uint64(second) << secondBitFieldOffset) & secondBitFieldMask
}

// Second returns the second value.
func (t CoreTime) Second() int {
	return int(t.getSecond())
}

func (t CoreTime) getMicrosecond() uint32 {
	return uint32((uint64(t) & microsecondBitFieldMask) >> microsecondBitFieldOffset)
}

func (t *CoreTime) setMicrosecond(microsecond uint32) {
	*(*uint64)(t) &= ^microsecondBitFieldMask
	*(*uint64)(t) |= (uint64(microsecond) << microsecondBitFieldOffset) & microsecondBitFieldMask
}

// Microsecond returns the microsecond value.
func (t CoreTime) Microsecond() int {
	return int(t.getMicrosecond())
}

// Weekday returns weekday value.
func (t CoreTime) Weekday() gotime.Weekday {
	// No need to consider timezone, use the date directly.
	t1, err := t.GoTime(gotime.UTC)
	// allow invalid dates
	if err != nil {
		return t1.Weekday()
	}
	return t1.Weekday()
}

// YearWeek returns year and week.
func (t CoreTime) YearWeek(mode int) (int, int) {
	behavior := weekMode(mode) | weekBehaviourYear
	return calcWeek(t, behavior)
}

// Week returns week value.
func (t CoreTime) Week(mode int) int {
	if t.getMonth() == 0 || t.getDay() == 0 {
		return 0
	}
	_, week := calcWeek(t, weekMode(mode))
	return week
}

// YearDay returns year and day.
func (t CoreTime) YearDay() int {
	if t.getMonth() == 0 || t.getDay() == 0 {
		return 0
	}
	year, month, day := t.Year(), t.Month(), t.Day()
	return calcDaynr(year, month, day) -
		calcDaynr(year, 1, 1) + 1
}

// GoTime converts Time to GoTime.
func (t CoreTime) GoTime(loc *gotime.Location) (gotime.Time, error) {
	// gotime.Time can't represent month 0 or day 0, date contains 0 would be converted to a nearest date,
	// For example, 2006-12-00 00:00:00 would become 2015-11-30 23:59:59.
	year, month, day, hour, minute, second, microsecond := t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Microsecond()
	tm := gotime.Date(year, gotime.Month(month), day, hour, minute, second, microsecond*1000, loc)
	year2, month2, day2 := tm.Date()
	hour2, minute2, second2 := tm.Clock()
	microsec2 := tm.Nanosecond() / 1000
	// This function will check the result, and return an error if it's not the same with the origin input .
	if year2 != year || int(month2) != month || day2 != day ||
		hour2 != hour || minute2 != minute || second2 != second ||
		microsec2 != microsecond {
		return tm, errors.New("go time not match")
	}
	return tm, nil
}

// IsLeapYear returns if it's leap year.
func (t CoreTime) IsLeapYear() bool {
	return isLeapYear(t.getYear())
}

func isLeapYear(year uint16) bool {
	return (year%4 == 0 && year%100 != 0) || year%400 == 0
}

var daysByMonth = [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// GetLastDay returns the last day of the month
func GetLastDay(year, month int) int {
	var day = 0
	if month > 0 && month <= 12 {
		day = daysByMonth[month-1]
	}
	if month == 2 && isLeapYear(uint16(year)) {
		day = 29
	}
	return day
}

func getFixDays(year, month, day int, ot gotime.Time) int {
	if (year != 0 || month != 0) && day == 0 {
		od := ot.Day()
		t := ot.AddDate(year, month, day)
		td := t.Day()
		if od != td {
			tm := int(t.Month()) - 1
			tMax := GetLastDay(t.Year(), tm)
			dd := tMax - od
			return dd
		}
	}
	return 0
}

// compareTime compare two Time.
// return:
//  0: if a == b
//  1: if a > b
// -1: if a < b
func compareTime(a, b CoreTime) int {
	ta := datetimeToUint64(a)
	tb := datetimeToUint64(b)

	switch {
	case ta < tb:
		return -1
	case ta > tb:
		return 1
	}

	switch {
	case a.Microsecond() < b.Microsecond():
		return -1
	case a.Microsecond() > b.Microsecond():
		return 1
	}

	return 0
}

// datetimeToUint64 converts time value to integer in YYYYMMDDHHMMSS format.
func datetimeToUint64(t CoreTime) uint64 {
	return uint64(t.Year())*1e10 +
		uint64(t.Month())*1e8 +
		uint64(t.Day())*1e6 +
		uint64(t.Hour())*1e4 +
		uint64(t.Minute())*1e2 +
		uint64(t.Second())
}

// AddDate fix gap between mysql and golang api
// When we execute select date_add('2018-01-31',interval 1 month) in mysql we got 2018-02-28
// but in tidb we got 2018-03-03.
// Dig it and we found it's caused by golang api time.Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location) Time ,
// it says October 32 converts to November 1 ,it conflicts with mysql.
// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_date-add
func AddDate(year, month, day int64, ot gotime.Time) (nt gotime.Time) {
	df := getFixDays(int(year), int(month), int(day), ot)
	if df != 0 {
		nt = ot.AddDate(int(year), int(month), df)
	} else {
		nt = ot.AddDate(int(year), int(month), int(day))
	}
	return nt
}

func calcTimeFromSec(to *CoreTime, seconds, microseconds int) {
	to.setHour(uint8(seconds / 3600))
	seconds = seconds % 3600
	to.setMinute(uint8(seconds / 60))
	to.setSecond(uint8(seconds % 60))
	to.setMicrosecond(uint32(microseconds))
}

const secondsIn24Hour = 86400

func calcTimeDiffInternal(t1 CoreTime, year, month, day, hour, minute, second, microsecond, sign int) (seconds, microseconds int, neg bool) {
	days := calcDaynr(t1.Year(), t1.Month(), t1.Day())
	days2 := calcDaynr(year, month, day)
	days -= sign * days2

	tmp := (int64(days)*secondsIn24Hour+
		int64(t1.Hour())*3600+int64(t1.Minute())*60+
		int64(t1.Second())-
		int64(sign)*(int64(hour)*3600+int64(minute)*60+
			int64(second)))*
		1e6 +
		int64(t1.Microsecond()) - int64(sign)*int64(microsecond)

	if tmp < 0 {
		tmp = -tmp
		neg = true
	}
	seconds = int(tmp / 1e6)
	microseconds = int(tmp % 1e6)
	return
}

// calcDaynr calculates days since 0000-00-00.
func calcDaynr(year, month, day int) int {
	if year == 0 && month == 0 {
		return 0
	}

	delsum := 365*year + 31*(month-1) + day
	if month <= 2 {
		year--
	} else {
		delsum -= (month*4 + 23) / 10
	}
	temp := ((year/100 + 1) * 3) / 4
	return delsum + year/4 - temp
}

// DateDiff calculates number of days between two days.
func DateDiff(startTime, endTime CoreTime) int {
	return calcDaynr(startTime.Year(), startTime.Month(), startTime.Day()) - calcDaynr(endTime.Year(), endTime.Month(), endTime.Day())
}

// calcDaysInYear calculates days in one year, it works with 0 <= year <= 99.
func calcDaysInYear(year int) int {
	if (year&3) == 0 && (year%100 != 0 || (year%400 == 0 && (year != 0))) {
		return 366
	}
	return 365
}

// calcWeekday calculates weekday from daynr, returns 0 for Monday, 1 for Tuesday ...
func calcWeekday(daynr int, sundayFirstDayOfWeek bool) int {
	daynr += 5
	if sundayFirstDayOfWeek {
		daynr++
	}
	return daynr % 7
}

type weekBehaviour uint

const (
	// weekBehaviourMondayFirst set Monday as first day of week; otherwise Sunday is first day of week
	weekBehaviourMondayFirst weekBehaviour = 1 << iota
	// If set, Week is in range 1-53, otherwise Week is in range 0-53.
	// Note that this flag is only relevant if WEEK_JANUARY is not set.
	weekBehaviourYear
	// If not set, Weeks are numbered according to ISO 8601:1988.
	// If set, the week that contains the first 'first-day-of-week' is week 1.
	weekBehaviourFirstWeekday
)

func (v weekBehaviour) test(flag weekBehaviour) bool {
	return (v & flag) != 0
}

func weekMode(mode int) weekBehaviour {
	weekFormat := weekBehaviour(mode & 7)
	if (weekFormat & weekBehaviourMondayFirst) == 0 {
		weekFormat ^= weekBehaviourFirstWeekday
	}
	return weekFormat
}

// calcWeek calculates week and year for the time.
func calcWeek(t CoreTime, wb weekBehaviour) (year int, week int) {
	var days int
	ty, tm, td := int(t.getYear()), int(t.getMonth()), int(t.getDay())
	daynr := calcDaynr(ty, tm, td)
	firstDaynr := calcDaynr(ty, 1, 1)
	mondayFirst := wb.test(weekBehaviourMondayFirst)
	weekYear := wb.test(weekBehaviourYear)
	firstWeekday := wb.test(weekBehaviourFirstWeekday)

	weekday := calcWeekday(firstDaynr, !mondayFirst)

	year = ty

	if tm == 1 && td <= 7-weekday {
		if !weekYear &&
			((firstWeekday && weekday != 0) || (!firstWeekday && weekday >= 4)) {
			week = 0
			return
		}
		weekYear = true
		year--
		days = calcDaysInYear(year)
		firstDaynr -= days
		weekday = (weekday + 53*7 - days) % 7
	}

	if (firstWeekday && weekday != 0) ||
		(!firstWeekday && weekday >= 4) {
		days = daynr - (firstDaynr + 7 - weekday)
	} else {
		days = daynr - (firstDaynr - weekday)
	}

	if weekYear && days >= 52*7 {
		weekday = (weekday + calcDaysInYear(year)) % 7
		if (!firstWeekday && weekday < 4) ||
			(firstWeekday && weekday == 0) {
			year++
			week = 1
			return
		}
	}
	week = days/7 + 1
	return
}
