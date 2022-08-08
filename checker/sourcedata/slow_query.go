// Copyright 2019 PingCAP, Inc.
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

package sourcedata

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pingcap/diag/pkg/types"
	"github.com/pingcap/diag/pkg/utils/hack"
	"github.com/pingcap/errors"
	"github.com/pingcap/log"
	"go.uber.org/zap"
)

// ParseSlowLogBatchSize is the batch size of slow-log lines for a worker to parse, exported for testing.
var ParseSlowLogBatchSize = 64
var MaxOfMaxAllowedPacket uint64 = 1073741824

const (
	// SlowLogTimeFormat is the time format for slow log.
	SlowLogTimeFormat = time.RFC3339Nano
	// OldSlowLogTimeFormat is the first version of the the time format for slow log, This is use for compatibility.
	OldSlowLogTimeFormat = "2006-01-02-15:04:05.999999999 -0700"
)

const maxReadCacheSize = 1024 * 1024 * 64

type slowQueryColumnValueFactory func(row []string, value string, tz *time.Location, checker *slowLogChecker) (valid bool, err error)

// slowQueryRetriever is used to read slow log data.
type slowQueryRetriever struct {
	concurrency int
	timeZone    *time.Location

	// origin field
	initialized           bool
	outputCols            []string
	files                 []logFile
	fileIdx               int
	fileLine              int
	checker               *slowLogChecker
	columnValueFactoryMap map[string]slowQueryColumnValueFactory

	taskList chan slowLogTask
	stats    *slowQueryRuntimeStats

	desc             bool
	filterTimeRanges bool
	timeRanges       []slowLogTimeRange
	singleFile       bool
	slowQueryFile    string
}

type slowLogTimeRange struct {
	StartTime time.Time
	EndTime   time.Time
}

type SlowQueryRetrieverOpt func(retriever *slowQueryRetriever) error

func WithTimeRanges(start, end time.Time) SlowQueryRetrieverOpt {
	tr := slowLogTimeRange{
		StartTime: start,
		EndTime:   end,
	}

	return func(retriever *slowQueryRetriever) error {
		retriever.filterTimeRanges = true
		if len(retriever.timeRanges) == 0 {
			retriever.timeRanges = []slowLogTimeRange{
				tr,
			}
		} else {
			retriever.timeRanges = append(retriever.timeRanges, tr)
		}
		return nil
	}
}

func NewSlowQueryRetriever(concurrency int, location *time.Location, cols []string, slowLogPath string, opts ...SlowQueryRetrieverOpt) (*slowQueryRetriever, error) {
	retriever := &slowQueryRetriever{
		concurrency:   concurrency,
		timeZone:      location,
		outputCols:    cols,
		singleFile:    false,
		slowQueryFile: slowLogPath,
	}
	for _, opt := range opts {
		if err := opt(retriever); err != nil {
			return nil, err
		}
	}
	return retriever, nil
}

func (e *slowQueryRetriever) retrieve(ctx context.Context) ([][]string, error) {
	if !e.initialized {
		err := e.initialize(ctx)
		if err != nil {
			return nil, err
		}
		e.initializeAsyncParsing(ctx)
	}
	return e.dataForSlowLog(ctx)
}

func (e *slowQueryRetriever) initialize(ctx context.Context) error {
	var err error
	// initialize column value factories.
	e.columnValueFactoryMap = make(map[string]slowQueryColumnValueFactory, len(e.outputCols))
	for idx, col := range e.outputCols {
		factory, err := getColumnValueFactoryByName(col, idx)
		if err != nil {
			return err
		}
		if factory == nil {
			panic(fmt.Sprintf("should never happen, should register new column %v into getColumnValueFactoryByName function", col))
		}
		e.columnValueFactoryMap[col] = factory
	}
	// initialize checker.
	e.checker = &slowLogChecker{}
	e.stats = &slowQueryRuntimeStats{}
	if e.filterTimeRanges {
		for _, tr := range e.timeRanges {
			startTime := types.NewTime(types.FromGoTime(tr.StartTime), 12, types.MaxFsp)
			endTime := types.NewTime(types.FromGoTime(tr.EndTime), 12, types.MaxFsp)
			timeRange := &timeRange{
				startTime: startTime,
				endTime:   endTime,
			}
			e.checker.timeRanges = append(e.checker.timeRanges, timeRange)
		}
	}
	e.initialized = true
	e.files, err = e.getAllFiles(ctx, e.slowQueryFile)
	if e.desc {
		e.reverseLogFiles()
	}
	return err
}

func (e *slowQueryRetriever) initializeAsyncParsing(ctx context.Context) {
	e.taskList = make(chan slowLogTask, 1)
	go e.parseDataForSlowLog(ctx)
}

func (e *slowQueryRetriever) parseDataForSlowLog(ctx context.Context) {
	file := e.getNextFile()
	if file == nil {
		close(e.taskList)
		return
	}
	reader := bufio.NewReader(file)
	e.parseSlowLog(ctx, reader, ParseSlowLogBatchSize)
}

func (e *slowQueryRetriever) dataForSlowLog(ctx context.Context) ([][]string, error) {
	var (
		task slowLogTask
		ok   bool
	)
	for {
		select {
		case task, ok = <-e.taskList:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if !ok {
			return nil, nil
		}
		result := <-task.resultCh
		rows, err := result.rows, result.err
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			continue
		}
		return rows, nil
	}
}

func (e *slowQueryRetriever) getBatchLog(ctx context.Context, reader *bufio.Reader, offset *offset, num int) ([][]string, error) {
	var line string
	log := make([]string, 0, num)
	var err error
	for i := 0; i < num; i++ {
		for {
			if isCtxDone(ctx) {
				return nil, ctx.Err()
			}
			e.fileLine++
			lineByte, err := getOneLine(reader)
			if err != nil {
				if err == io.EOF {
					e.fileLine = 0
					file := e.getNextFile()
					if file == nil {
						return [][]string{log}, nil
					}
					offset.length = len(log)
					reader.Reset(file)
					continue
				}
				return [][]string{log}, err
			}
			line = string(hack.String(lineByte))
			log = append(log, line)
			if strings.HasSuffix(line, SlowLogSQLSuffixStr) {
				if strings.HasPrefix(line, "use") || strings.HasPrefix(line, SlowLogRowPrefixStr) {
					continue
				}
				break
			}
		}
	}
	return [][]string{log}, err
}

func (e *slowQueryRetriever) getBatchLogForReversedScan(ctx context.Context, reader *bufio.Reader, offset *offset, num int) ([][]string, error) {
	// reader maybe change when read previous file.
	inputReader := reader
	defer func() {
		file := e.getNextFile()
		if file != nil {
			inputReader.Reset(file)
		}
	}()
	var line string
	var logs []slowLogBlock
	var log []string
	var err error
	hasStartFlag := false
	scanPreviousFile := false
	for {
		if isCtxDone(ctx) {
			return nil, ctx.Err()
		}
		e.fileLine++
		lineByte, err := getOneLine(reader)
		if err != nil {
			if err == io.EOF {
				if len(log) == 0 {
					decomposedSlowLogTasks := decomposeToSlowLogTasks(logs, num)
					offset.length = len(decomposedSlowLogTasks)
					return decomposedSlowLogTasks, nil
				}
				e.fileLine = 0
				file := e.getPreviousFile()
				if file == nil {
					return decomposeToSlowLogTasks(logs, num), nil
				}
				reader = bufio.NewReader(file)
				scanPreviousFile = true
				continue
			}
			return nil, err
		}
		line = string(hack.String(lineByte))
		if !hasStartFlag && strings.HasPrefix(line, SlowLogStartPrefixStr) {
			hasStartFlag = true
		}
		if hasStartFlag {
			log = append(log, line)
			if strings.HasSuffix(line, SlowLogSQLSuffixStr) {
				if strings.HasPrefix(line, "use") || strings.HasPrefix(line, SlowLogRowPrefixStr) {
					continue
				}
				logs = append(logs, log)
				if scanPreviousFile {
					break
				}
				log = make([]string, 0, 8)
				hasStartFlag = false
			}
		}
	}
	return decomposeToSlowLogTasks(logs, num), err
}

func (e *slowQueryRetriever) getNextFile() *os.File {
	if e.fileIdx >= len(e.files) {
		return nil
	}
	file := e.files[e.fileIdx].file
	e.fileIdx++
	if e.stats != nil {
		stat, err := file.Stat()
		if err == nil {
			// ignore the err will be ok.
			e.stats.readFileSize += stat.Size()
			e.stats.readFileNum++
		}
	}
	return file
}

func (e *slowQueryRetriever) getPreviousFile() *os.File {
	fileIdx := e.fileIdx
	// fileIdx refer to the next file which should be read
	// so we need to set fileIdx to fileIdx - 2 to get the previous file.
	fileIdx = fileIdx - 2
	if fileIdx < 0 {
		return nil
	}
	file := e.files[fileIdx].file
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil
	}
	return file
}

func (e *slowQueryRetriever) getAllFiles(ctx context.Context, logFilePath string) ([]logFile, error) {
	totalFileNum := 0
	if e.stats != nil {
		startTime := time.Now()
		defer func() {
			e.stats.initialize = time.Since(startTime)
			e.stats.totalFileNum = totalFileNum
		}()
	}
	if e.singleFile {
		totalFileNum = 1
		file, err := os.Open(logFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		return []logFile{{file: file}}, nil
	}
	var logFiles []logFile
	logDir := filepath.Dir(logFilePath)
	ext := filepath.Ext(logFilePath)
	prefix := logFilePath[:len(logFilePath)-len(ext)]
	handleErr := func(err error) error {
		// Ignore the error and append warning for usability.
		if err != io.EOF {
			log.Warn("get slow log file err", zap.Error(err))
		}
		return nil
	}
	files, err := os.ReadDir(logDir)
	if err != nil {
		return nil, err
	}
	walkFn := func(path string, info os.DirEntry) error {
		if info.IsDir() {
			return nil
		}
		// All rotated log files have the same prefix with the original file.
		if !strings.HasPrefix(path, prefix) {
			return nil
		}
		if isCtxDone(ctx) {
			return ctx.Err()
		}
		totalFileNum++
		file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return handleErr(err)
		}
		skip := false
		defer func() {
			if !skip {
				log.Error("defer error", zap.Error(file.Close()))
			}
		}()
		// Get the file start time.
		fileStartTime, err := e.getFileStartTime(ctx, file)
		if err != nil {
			return handleErr(err)
		}
		start := types.NewTime(types.FromGoTime(fileStartTime), 12, types.MaxFsp)
		notInAllTimeRanges := true
		for _, tr := range e.checker.timeRanges {
			if start.Compare(tr.endTime) <= 0 {
				notInAllTimeRanges = false
				break
			}
		}
		if notInAllTimeRanges {
			return nil
		}

		// Get the file end time.
		fileEndTime, err := e.getFileEndTime(ctx, file)
		if err != nil {
			return handleErr(err)
		}
		end := types.NewTime(types.FromGoTime(fileEndTime), 12, types.MaxFsp)
		inTimeRanges := false
		for _, tr := range e.checker.timeRanges {
			if !(start.Compare(tr.endTime) > 0 || end.Compare(tr.startTime) < 0) {
				inTimeRanges = true
				break
			}
		}
		if !inTimeRanges {
			return nil
		}
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return handleErr(err)
		}
		logFiles = append(logFiles, logFile{
			file:  file,
			start: fileStartTime,
			end:   fileEndTime,
		})
		skip = true
		return nil
	}
	for _, file := range files {
		err := walkFn(filepath.Join(logDir, file.Name()), file)
		if err != nil {
			return nil, err
		}
	}
	// Sort by start time
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].start.Before(logFiles[j].start)
	})
	return logFiles, err
}

func (e *slowQueryRetriever) getFileStartTime(ctx context.Context, file *os.File) (time.Time, error) {
	var t time.Time
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return t, err
	}
	reader := bufio.NewReader(file)
	maxNum := 128
	for {
		lineByte, err := getOneLine(reader)
		if err != nil {
			return t, err
		}
		line := string(lineByte)
		if strings.HasPrefix(line, SlowLogStartPrefixStr) {
			return ParseTime(line[len(SlowLogStartPrefixStr):])
		}
		maxNum -= 1
		if maxNum <= 0 {
			break
		}
		if isCtxDone(ctx) {
			return t, ctx.Err()
		}
	}
	return t, errors.Errorf("malform slow query file %v", file.Name())
}

func (e *slowQueryRetriever) getFileEndTime(ctx context.Context, file *os.File) (time.Time, error) {
	var t time.Time
	var tried int
	stat, err := file.Stat()
	if err != nil {
		return t, err
	}
	endCursor := stat.Size()
	maxLineNum := 128
	for {
		lines, readBytes, err := readLastLines(ctx, file, endCursor)
		if err != nil {
			return t, err
		}
		// read out the file
		if readBytes == 0 {
			break
		}
		endCursor -= int64(readBytes)
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.HasPrefix(lines[i], SlowLogStartPrefixStr) {
				return ParseTime(lines[i][len(SlowLogStartPrefixStr):])
			}
		}
		tried += len(lines)
		if tried >= maxLineNum {
			break
		}
		if isCtxDone(ctx) {
			return t, ctx.Err()
		}
	}
	return t, errors.Errorf("invalid slow query file %v", file.Name())
}

func (e *slowQueryRetriever) getRuntimeStats() RuntimeStats {
	return e.stats
}

func (e *slowQueryRetriever) reverseLogFiles() {
	for i := 0; i < len(e.files)/2; i++ {
		j := len(e.files) - i - 1
		e.files[i], e.files[j] = e.files[j], e.files[i]
	}
}

func (e *slowQueryRetriever) Close() error {
	for _, f := range e.files {
		err := f.file.Close()
		if err != nil {
			log.Error("close slow log file failed.", zap.Error(err))
		}
	}
	return nil
}

func (e *slowQueryRetriever) sendParsedSlowLogCh(ctx context.Context, t slowLogTask, re parsedSlowLog) {
	select {
	case t.resultCh <- re:
	case <-ctx.Done():
		return
	}
}

func (e *slowQueryRetriever) parseSlowLog(ctx context.Context, reader *bufio.Reader, logNum int) {
	defer close(e.taskList)
	var wg sync.WaitGroup
	offset := offset{offset: 0, length: 0}
	// To limit the num of go routine
	concurrent := e.concurrency
	ch := make(chan int, concurrent)
	if e.stats != nil {
		e.stats.concurrent = concurrent
	}
	defer close(ch)
	for {
		startTime := time.Now()
		var logs [][]string
		var err error
		if !e.desc {
			logs, err = e.getBatchLog(ctx, reader, &offset, logNum)
		} else {
			logs, err = e.getBatchLogForReversedScan(ctx, reader, &offset, logNum)
		}
		if err != nil {
			t := slowLogTask{}
			t.resultCh = make(chan parsedSlowLog, 1)
			select {
			case <-ctx.Done():
				return
			case e.taskList <- t:
			}
			e.sendParsedSlowLogCh(ctx, t, parsedSlowLog{nil, err})
		}
		if len(logs) == 0 || len(logs[0]) == 0 {
			break
		}
		if e.stats != nil {
			e.stats.readFile += time.Since(startTime)
		}
		for i := range logs {
			log := logs[i]
			t := slowLogTask{}
			t.resultCh = make(chan parsedSlowLog, 1)
			start := offset
			wg.Add(1)
			ch <- 1
			e.taskList <- t
			go func() {
				defer wg.Done()
				result, err := e.parseLog(ctx, log, start)
				e.sendParsedSlowLogCh(ctx, t, parsedSlowLog{result, err})
				<-ch
			}()
			offset.offset = e.fileLine
			offset.length = 0
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}
	wg.Wait()
}

func (e *slowQueryRetriever) parseLog(ctx context.Context, logs []string, offset offset) (data [][]string, err error) {
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
			buf := make([]byte, 4096)
			stackSize := runtime.Stack(buf, false)
			buf = buf[:stackSize]
			log.Warn("slow query parse slow log panic", zap.Error(err), zap.String("stack", string(buf)))
		}
		if e.stats != nil {
			atomic.AddInt64(&e.stats.parseLog, int64(time.Since(start)))
		}
	}()
	var row []string
	tz := e.timeZone
	startFlag := false
	for index, line := range logs {
		if isCtxDone(ctx) {
			return nil, ctx.Err()
		}
		fileLine := getLineIndex(offset, index)
		if !startFlag && strings.HasPrefix(line, SlowLogStartPrefixStr) {
			row = make([]string, len(e.outputCols))
			valid := e.setColumnValue(row, tz, SlowLogTimeStr, line[len(SlowLogStartPrefixStr):], e.checker, fileLine)
			if valid {
				startFlag = true
			}
			continue
		}
		if startFlag {
			if strings.HasPrefix(line, SlowLogRowPrefixStr) {
				line = line[len(SlowLogRowPrefixStr):]
				valid := true
				if strings.HasPrefix(line, SlowLogPrevStmtPrefix) {
					valid = e.setColumnValue(row, tz, SlowLogPrevStmt, line[len(SlowLogPrevStmtPrefix):], e.checker, fileLine)
				} else if strings.HasPrefix(line, SlowLogUserAndHostStr+SlowLogSpaceMarkStr) {
					value := line[len(SlowLogUserAndHostStr+SlowLogSpaceMarkStr):]
					fields := strings.SplitN(value, "@", 2)
					if len(fields) < 2 {
						continue
					}
					user := parseUserOrHostValue(fields[0])
					valid = e.setColumnValue(row, tz, SlowLogUserStr, user, e.checker, fileLine)
					if !valid {
						startFlag = false
						continue
					}
					host := parseUserOrHostValue(fields[1])
					valid = e.setColumnValue(row, tz, SlowLogHostStr, host, e.checker, fileLine)
				} else if strings.HasPrefix(line, SlowLogCopBackoffPrefix) {
					valid = e.setColumnValue(row, tz, SlowLogBackoffDetail, line, e.checker, fileLine)
				} else {
					fieldValues := strings.Split(line, " ")
					for i := 0; i < len(fieldValues)-1; i += 2 {
						field := strings.TrimSuffix(fieldValues[i], ":")
						valid := e.setColumnValue(row, tz, field, fieldValues[i+1], e.checker, fileLine)
						if !valid {
							startFlag = false
							break
						}
					}
				}
				if !valid {
					startFlag = false
				}
			} else if strings.HasSuffix(line, SlowLogSQLSuffixStr) {
				if strings.HasPrefix(line, "use") {
					// `use DB` statements in the slow log is used to keep it be compatible with MySQL,
					// since we already get the current DB from the `# DB` field, we can ignore it here,
					// please see https://github.com/pingcap/tidb/issues/17846 for more details.
					continue
				}
				// Get the sql string, and mark the start flag to false.
				_ = e.setColumnValue(row, tz, SlowLogQuerySQLStr, string(hack.Slice(line)), e.checker, fileLine)
				data = append(data, row)
				startFlag = false
			} else {
				startFlag = false
			}
		}
	}
	return data, nil
}

func (e *slowQueryRetriever) setColumnValue(row []string, tz *time.Location, field, value string, checker *slowLogChecker, _ int) bool {
	factory := e.columnValueFactoryMap[field]
	if factory == nil {
		return true
	}
	valid, err := factory(row, value, tz, checker)
	if err != nil {
		return true
	}
	return valid
}

// TODO: use string instead of Datum
type parsedSlowLog struct {
	rows [][]string
	err  error
}

type offset struct {
	offset int
	length int
}

type slowLogTask struct {
	resultCh chan parsedSlowLog
}

type slowLogBlock []string

type logFile struct {
	file       *os.File  // The opened file handle
	start, end time.Time // The start/end time of the log file
}

// RuntimeStats is used to express the executor runtime information.
type RuntimeStats interface {
	String() string
	Merge(RuntimeStats)
	Clone() RuntimeStats
	Tp() int
}

type slowQueryRuntimeStats struct {
	totalFileNum int
	readFileNum  int
	readFile     time.Duration
	initialize   time.Duration
	readFileSize int64
	parseLog     int64
	concurrent   int
}

// String implements the RuntimeStats interface.
func (s *slowQueryRuntimeStats) String() string {
	return fmt.Sprintf("initialize: %s, read_file: %s, parse_log: {time:%s, concurrency:%v}, total_file: %v, read_file: %v, read_size: %s",
		FormatDuration(s.initialize), FormatDuration(s.readFile),
		FormatDuration(time.Duration(s.parseLog)), s.concurrent,
		s.totalFileNum, s.readFileNum, FormatBytes(s.readFileSize))
}

// Merge implements the RuntimeStats interface.
func (s *slowQueryRuntimeStats) Merge(rs RuntimeStats) {
	tmp, ok := rs.(*slowQueryRuntimeStats)
	if !ok {
		return
	}
	s.totalFileNum += tmp.totalFileNum
	s.readFileNum += tmp.readFileNum
	s.readFile += tmp.readFile
	s.initialize += tmp.initialize
	s.readFileSize += tmp.readFileSize
	s.parseLog += tmp.parseLog
}

// Clone implements the RuntimeStats interface.
func (s *slowQueryRuntimeStats) Clone() RuntimeStats {
	newRs := *s
	return &newRs
}

// Tp implements the RuntimeStats interface.
func (s *slowQueryRuntimeStats) Tp() int {
	return TpSlowQueryRuntimeStat
}

type timeRange struct {
	startTime types.Time
	endTime   types.Time
}

type slowLogChecker struct {
	// Below fields is used to check privilege.
	// Below fields is used to check slow log time valid.
	enableTimeCheck bool
	timeRanges      []*timeRange
}

func (sc *slowLogChecker) isTimeValid(t types.Time) bool {
	for _, tr := range sc.timeRanges {
		if sc.enableTimeCheck && (t.Compare(tr.startTime) >= 0 && t.Compare(tr.endTime) <= 0) {
			return true
		}
	}
	return !sc.enableTimeCheck
}

// helper function
func getColumnValueFactoryByName(colName string, columnIdx int) (slowQueryColumnValueFactory, error) {
	switch colName {
	case SlowLogTimeStr:
		return func(row []string, value string, tz *time.Location, checker *slowLogChecker) (bool, error) {
			t, err := ParseTime(value)
			if err != nil {
				return false, err
			}
			timeValue := types.NewTime(types.FromGoTime(t), 12, types.MaxFsp)
			if checker != nil {
				valid := checker.isTimeValid(timeValue)
				if !valid {
					return valid, nil
				}
			}
			if t.Location() != tz {
				t = t.In(tz)
			}
			row[columnIdx] = t.Format("2006-01-02 15:04:05.999999")
			return true, nil
		}, nil
	case SlowLogBackoffDetail:
		return func(row []string, value string, tz *time.Location, checker *slowLogChecker) (bool, error) {
			backoffDetail := row[columnIdx]
			if len(backoffDetail) > 0 {
				backoffDetail += " "
			}
			backoffDetail += value
			row[columnIdx] = backoffDetail
			return true, nil
		}, nil
	case SlowLogPlan:
		return func(row []string, value string, tz *time.Location, checker *slowLogChecker) (bool, error) {
			plan := parsePlan(value)
			row[columnIdx] = plan
			return true, nil
		}, nil
	default:
		return func(row []string, value string, tz *time.Location, checker *slowLogChecker) (valid bool, err error) {
			row[columnIdx] = value
			return true, nil
		}, nil
	}
}

func getOneLine(reader *bufio.Reader) ([]byte, error) {
	var resByte []byte
	lineByte, isPrefix, err := reader.ReadLine()
	if isPrefix {
		// Need to read more data.
		resByte = make([]byte, len(lineByte), len(lineByte)*2)
	} else {
		resByte = make([]byte, len(lineByte))
	}
	// Use copy here to avoid shallow copy problem.
	copy(resByte, lineByte)
	if err != nil {
		return resByte, err
	}

	var tempLine []byte
	for isPrefix {
		tempLine, isPrefix, err = reader.ReadLine()
		resByte = append(resByte, tempLine...)
		// Use the max value of max_allowed_packet to check the single line length.
		if len(resByte) > int(MaxOfMaxAllowedPacket) {
			return resByte, errors.Errorf("single line length exceeds limit: %v", MaxOfMaxAllowedPacket)
		}
		if err != nil {
			return resByte, err
		}
	}
	return resByte, err
}

func getLineIndex(offset offset, index int) int {
	var fileLine int
	if offset.length <= index {
		fileLine = index - offset.length + 1
	} else {
		fileLine = offset.offset + index + 1
	}
	return fileLine
}

func isCtxDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func decomposeToSlowLogTasks(logs []slowLogBlock, num int) [][]string {
	if len(logs) == 0 {
		return nil
	}

	//In reversed scan, We should reverse the blocks.
	last := len(logs) - 1
	for i := 0; i < len(logs)/2; i++ {
		logs[i], logs[last-i] = logs[last-i], logs[i]
	}

	decomposedSlowLogTasks := make([][]string, 0)
	log := make([]string, 0, num*len(logs[0]))
	for i := range logs {
		log = append(log, logs[i]...)
		if i > 0 && i%num == 0 {
			decomposedSlowLogTasks = append(decomposedSlowLogTasks, log)
			log = make([]string, 0, len(log))
		}
	}
	if len(log) > 0 {
		decomposedSlowLogTasks = append(decomposedSlowLogTasks, log)
	}
	return decomposedSlowLogTasks
}

func parseUserOrHostValue(value string) string {
	// the new User&Host format: root[root] @ localhost [127.0.0.1]
	tmp := strings.Split(value, "[")
	return strings.TrimSpace(tmp[0])
}

func parsePlan(planString string) string {
	if len(planString) <= len(SlowLogPlanPrefix)+len(SlowLogPlanSuffix) {
		return planString
	}
	planString = planString[len(SlowLogPlanPrefix) : len(planString)-len(SlowLogPlanSuffix)]
	return planString
}

// ParseTime exports for testing.
func ParseTime(s string) (time.Time, error) {
	t, err := time.Parse(SlowLogTimeFormat, s)
	if err != nil {
		// This is for compatibility.
		t, err = time.Parse(OldSlowLogTimeFormat, s)
		if err != nil {
			err = errors.Errorf("string \"%v\" doesn't has a prefix that matches format \"%v\", err: %v", s, SlowLogTimeFormat, err)
		}
	}
	return t, err
}

// Read lines from the end of a file
// endCursor initial value should be the filesize
func readLastLines(ctx context.Context, file *os.File, endCursor int64) ([]string, int, error) {
	var lines []byte
	var firstNonNewlinePos int
	var cursor = endCursor
	var size int64 = 2048
	for {
		// stop if we are at the beginning
		// check it in the start to avoid read beyond the size
		if cursor <= 0 {
			break
		}
		if size < maxReadCacheSize {
			size = size * 2
		}
		if cursor < size {
			size = cursor
		}
		cursor -= size

		_, err := file.Seek(cursor, io.SeekStart)
		if err != nil {
			return nil, 0, err
		}
		chars := make([]byte, size)
		_, err = file.Read(chars)
		if err != nil {
			return nil, 0, err
		}
		lines = append(chars, lines...)

		// find first '\n' or '\r'
		for i := 0; i < len(chars); i++ {
			// reach the line end
			// the first newline may be in the line end at the first round
			if i >= len(lines)-1 {
				break
			}
			if (chars[i] == 10 || chars[i] == 13) && chars[i+1] != 10 && chars[i+1] != 13 {
				firstNonNewlinePos = i + 1
				break
			}
		}
		if firstNonNewlinePos > 0 {
			break
		}
		if isCtxDone(ctx) {
			return nil, 0, ctx.Err()
		}
	}
	finalStr := string(lines[firstNonNewlinePos:])
	return strings.Split(strings.ReplaceAll(finalStr, "\r\n", "\n"), "\n"), len(finalStr), nil
}

// FormatDuration uses to format duration, this function will prune precision before format duration.
// Pruning precision is for human readability. The prune rule is:
//  1. if the duration was less than 1us, return the original string.
//  2. readable value >=10, keep 1 decimal, otherwise, keep 2 decimal. such as:
//     9.412345ms  -> 9.41ms
//     10.412345ms -> 10.4ms
//     5.999s      -> 6s
//     100.45µs    -> 100.5µs
func FormatDuration(d time.Duration) string {
	if d <= time.Microsecond {
		return d.String()
	}
	unit := getUnit(d)
	if unit == time.Nanosecond {
		return d.String()
	}
	integer := (d / unit) * unit
	decimal := float64(d%unit) / float64(unit)
	if d < 10*unit {
		decimal = math.Round(decimal*100) / 100
	} else {
		decimal = math.Round(decimal*10) / 10
	}
	d = integer + time.Duration(decimal*float64(unit))
	return d.String()
}

func getUnit(d time.Duration) time.Duration {
	if d >= time.Second {
		return time.Second
	} else if d >= time.Millisecond {
		return time.Millisecond
	} else if d >= time.Microsecond {
		return time.Microsecond
	}
	return time.Nanosecond
}

const (
	byteSizeGB = int64(1 << 30)
	byteSizeMB = int64(1 << 20)
	byteSizeKB = int64(1 << 10)
	byteSizeBB = int64(1)
)

// BytesToString converts the memory consumption to a readable string.
func BytesToString(numBytes int64) string {
	GB := float64(numBytes) / float64(byteSizeGB)
	if GB > 1 {
		return fmt.Sprintf("%v GB", GB)
	}

	MB := float64(numBytes) / float64(byteSizeMB)
	if MB > 1 {
		return fmt.Sprintf("%v MB", MB)
	}

	KB := float64(numBytes) / float64(byteSizeKB)
	if KB > 1 {
		return fmt.Sprintf("%v KB", KB)
	}

	return fmt.Sprintf("%v Bytes", numBytes)
}

// FormatBytes uses to format bytes, this function will prune precision before format bytes.
func FormatBytes(numBytes int64) string {
	if numBytes <= byteSizeKB {
		return BytesToString(numBytes)
	}
	unit, unitStr := getByteUnit(numBytes)
	if unit == byteSizeBB {
		return BytesToString(numBytes)
	}
	v := float64(numBytes) / float64(unit)
	decimal := 1
	if numBytes%unit == 0 {
		decimal = 0
	} else if v < 10 {
		decimal = 2
	}
	return fmt.Sprintf("%v %s", strconv.FormatFloat(v, 'f', decimal, 64), unitStr)
}

func getByteUnit(b int64) (int64, string) {
	if b > byteSizeGB {
		return byteSizeGB, "GB"
	} else if b > byteSizeMB {
		return byteSizeMB, "MB"
	} else if b > byteSizeKB {
		return byteSizeKB, "KB"
	}
	return byteSizeBB, "Bytes"
}
