// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package scraper

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pingcap/diag/collector/log/parser"
)

const (
	seekLimit      = 1024 * 1024 * 1024 // 1MB
	LogTypeStd     = "std"
	LogTypeSlow    = "slow"
	LogTypeRocksDB = "rocksdb"
	LogTypeUnknown = "unknown"

	// maxLeadingLines bounds the scan for the first timestamped line of a log
	// file, which tells whether the file overlaps the requested range. TiKV
	// stamps every rocksdb line, so the first line normally matches already;
	// the scan only covers files starting with an unstamped line.
	maxLeadingLines = 64
)

func IsValidLogType(logtype string) bool {
	switch logtype {
	case LogTypeStd, LogTypeSlow, LogTypeRocksDB, LogTypeUnknown:
		return true
	default:
		return false
	}
}

// LogScraper scraps log files of components
type LogScraper struct {
	Paths   []string        // paths of log files
	Types   map[string]bool // log type
	Start   time.Time       // start time
	End     time.Time       // end time
	Trim    bool            // copy only the lines inside [Start, End]
	TrimDir string          // where the trimmed copies are written, required by Trim
}

// Scrap implements the Scraper interface
func (s *LogScraper) Scrap(result *Sample) error {
	if s.Trim && s.TrimDir == "" {
		return fmt.Errorf("trim dir is required when trimming is enabled")
	}
	if result.Log == nil {
		result.Log = make(FileStat)
	}
	if result.LogTypes == nil {
		result.LogTypes = make(FileTypes)
	}
	fileList := make([]string, 0)

	// extend all file paths
	for _, fp := range s.Paths {
		if fm, err := filepath.Glob(fp); err == nil {
			fileList = append(fileList, fm...)
		} else {
			fmt.Fprintf(os.Stderr, "error scrapping %s: %s\n", fp, err)
			continue
		}
	}

	// filter log files
	for _, fp := range fileList {
		fi, err := os.Stat(fp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error checking %s: %s\n", fp, err)
			continue
		}
		if fi.IsDir() {
			continue
		}

		logtype, in, err := getLogType(fp, fi, s.Start, s.End)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error checking %s: %s\n", fp, err)
			continue
		}
		if !s.Types[logtype] || !in {
			continue
		}

		// The scraper reports the paths to download, so a trimmed copy simply
		// takes the place of the original file in the result.
		target, size := fp, fi.Size()
		if s.Trim {
			trimmed, trimmedSize, err := s.trimFile(fp, logtype)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error trimming %s: %s, collecting the whole file\n", fp, err)
			} else if trimmed != "" {
				target, size = trimmed, trimmedSize
			}
		}
		result.Log[target] = size
		result.LogTypes[target] = logtype
	}

	return nil
}

func getLogType(fpath string, fi fs.FileInfo, start, end time.Time) (logtype string, inrange bool, err error) {
	fileName := filepath.Base(fpath)
	// collect stderr log despite time range
	if strings.Contains(fileName, "stderr") {
		return LogTypeStd, true, nil
	}

	// rocksdb lines use a timestamp format of their own, see rocksDBTimeRE, so
	// they cannot go through the parser based check below.
	if strings.HasPrefix(fileName, "rocksdb") && strings.HasSuffix(fileName, ".info") {
		return LogTypeRocksDB, fileHeadInRange(fpath, fi, rocksDBLineTime, start, end), nil
	}

	f, err := os.Open(fpath)
	if err != nil {
		return "", false, err
	}
	defer f.Close()

	var r io.ReadCloser = f
	if strings.HasSuffix(fpath, ".gz") {
		r, err = gzip.NewReader(f)
		if err != nil {
			return LogTypeUnknown, false, err
		}
		defer r.Close()
	}

	bufr := bufio.NewReader(r)
	// read the first line of log file
	head, _, err := bufr.ReadLine()
	if err == nil {
		ht := parseLine(head, parser.ListStd())
		if ht != nil {
			if ht.After(end) || fi.ModTime().Before(start) {
				return LogTypeStd, false, nil
			}
			return LogTypeStd, true, nil
		}
		p := &parser.SlowQueryParser{}
		ht, _ = p.ParseHead(head)
		if ht != nil {
			if ht.After(end) || fi.ModTime().Before(start) {
				return LogTypeSlow, false, nil
			}
			return LogTypeSlow, true, nil
		}
	}

	// use create time as head time for unknown file
	// cTime := fi.Sys().(*syscall.Stat_t).Ctim
	// ht := time.Unix(int64(cTime.Sec), int64(cTime.Nsec))
	if fi.ModTime().Before(start) {
		return LogTypeUnknown, false, nil
	}
	return LogTypeUnknown, true, nil
}

func parseLine(line []byte, parsers []parser.Parser) *time.Time {
	for _, p := range parsers {
		if t, _ := p.ParseHead(line); t != nil {
			return t
		}
	}
	return nil
}

// rocksDBTimeRE matches the timestamp a rocksdb log line starts with. TiKV
// formats those lines on its own:
//
//	[2024/01/01 12:00:00.000 +08:00][12345][INFO] <message>
//
// It differs from a component log line ([ts] [LEVEL] [module]): the second
// bracket holds the thread id instead of the level, and the brackets are not
// separated by a blank, so the parsers of collector/log/parser cannot read it.
var rocksDBTimeRE = regexp.MustCompile(`^\[(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}\.\d{3} [+-]\d{2}:\d{2})\]`)

// stdParsers and slowQueryParser are built once: when trimming, every line of a
// file is parsed and parser.ListStd allocates a new slice on each call.
var (
	stdParsers     = parser.ListStd()
	slowQueryParse = &parser.SlowQueryParser{}
	jsonLogLine    = &parser.UnifiedJSONLogParser{}
)

// lineTime extracts the timestamp a log line starts with. It reports false for
// lines without one, which in a rocksdb log means a continuation line of a
// multi-line record: TiKV stamps the first physical line of a record only.
type lineTime func(line []byte) (time.Time, bool)

// lineBody strips the line ending. Extractors are fed lines as they come from
// the file, and the parsers of collector/log/parser anchor some patterns with
// "$", which only matches at the very end of the input: a slow query header
// would not be recognised while it still carries its "\n".
func lineBody(line []byte) []byte {
	return bytes.TrimRight(line, "\r\n")
}

func rocksDBLineTime(line []byte) (time.Time, bool) {
	if m := rocksDBTimeRE.FindSubmatch(line); m != nil {
		if t, err := time.Parse(parser.TimeStampLayout, string(m[1])); err == nil {
			return t, true
		}
	}
	// The very same logs go through TiKV's JSON formatter when
	// log.format = "json" (keys: time, level, caller, message), and a rocksdb
	// log file is recognised by its name, so the text shape can not be assumed.
	// The cheap first byte check keeps the JSON parser off the text path.
	if len(line) > 0 && line[0] == '{' {
		if t, _ := jsonLogLine.ParseHead(line); t != nil {
			return *t, true
		}
	}
	return time.Time{}, false
}

func stdLineTime(line []byte) (time.Time, bool) {
	if t := parseLine(line, stdParsers); t != nil {
		return *t, true
	}
	return time.Time{}, false
}

func slowLineTime(line []byte) (time.Time, bool) {
	if t, _ := slowQueryParse.ParseHead(line); t != nil {
		return *t, true
	}
	return time.Time{}, false
}

// lineTimeOf returns how to read the timestamp of a line of the given log type.
// A nil extractor means the type carries no per line timestamp and cannot be
// trimmed.
func lineTimeOf(logtype string) lineTime {
	switch logtype {
	case LogTypeRocksDB:
		return rocksDBLineTime
	case LogTypeStd:
		return stdLineTime
	case LogTypeSlow:
		return slowLineTime
	default:
		return nil
	}
}

// fileHeadInRange reports whether a log file overlaps [start, end]. It is the
// file level counterpart of trimToRange and avoids downloading files that lie
// entirely outside the range.
func fileHeadInRange(fpath string, fi fs.FileInfo, extract lineTime, start, end time.Time) bool {
	if fi.ModTime().Before(start) {
		// nothing was appended to the file after the range began
		return false
	}

	f, err := os.Open(fpath)
	if err != nil {
		return true // never drop a file that cannot be inspected
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 64*1024)
	for i := 0; i < maxLeadingLines; i++ {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			if t, ok := extract(lineBody(line)); ok {
				return !t.After(end)
			}
		}
		if err != nil {
			break
		}
	}
	// No timestamp at all, e.g. log.enable-timestamp = false. Keep the file
	// rather than dropping it silently.
	return true
}

// trimToRange copies the lines of src whose timestamp falls inside [start, end]
// into dst and returns the number of bytes written. A line without a timestamp
// inherits the decision of the previous timestamped line, so a multi-line
// record is never cut in half. found reports whether any timestamp was seen:
// when it is false the caller must keep the original file, otherwise a file
// with an unexpected layout would be replaced by an empty copy.
func trimToRange(src, dst string, extract lineTime, start, end time.Time) (written int64, found bool, err error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, false, err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return 0, false, err
	}
	out, err := os.Create(dst)
	if err != nil {
		return 0, false, err
	}
	defer out.Close()

	r := bufio.NewReaderSize(in, 64*1024)
	w := bufio.NewWriterSize(out, 64*1024)
	keep := true // a leading line without a timestamp is kept
	for {
		line, rerr := r.ReadBytes('\n')
		if len(line) > 0 {
			if t, ok := extract(lineBody(line)); ok {
				found = true
				keep = !t.Before(start) && !t.After(end)
			}
			if keep {
				n, werr := w.Write(line)
				written += int64(n)
				if werr != nil {
					return written, found, werr
				}
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			return written, found, rerr
		}
	}
	if err := w.Flush(); err != nil {
		return written, found, err
	}

	return written, found, nil
}

// trimFile writes a range trimmed copy of a log file below s.TrimDir and
// returns its path and size. An empty path means the file has to be collected
// as it is.
func (s *LogScraper) trimFile(src, logtype string) (string, int64, error) {
	extract := lineTimeOf(logtype)
	if extract == nil {
		return "", 0, nil // this log type carries no per line timestamp
	}

	dst := trimmedPath(s.TrimDir, src)
	written, found, err := trimToRange(src, dst, extract, s.Start, s.End)
	if err != nil {
		os.Remove(dst)
		return "", 0, err
	}
	if !found {
		// Not a single timestamp: keep the whole file instead of an empty one.
		os.Remove(dst)
		return "", 0, nil
	}
	return dst, written, nil
}

// trimmedPath maps a source file into the trim directory while keeping its full
// source path, so that files sharing a name do not collide: a host running
// several TiKV instances holds one rocksdb.info per data directory.
func trimmedPath(trimDir, src string) string {
	clean := filepath.Clean(src)
	if filepath.IsAbs(clean) {
		return filepath.Join(trimDir, strings.TrimPrefix(clean, string(filepath.Separator)))
	}
	// A relative source path cannot be mirrored safely, as it may escape the
	// trim directory, so it is flattened with a hash.
	return filepath.Join(trimDir, fmt.Sprintf("%s.%s", filepath.Base(clean), shortHash(clean)))
}

func shortHash(s string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%08x", h.Sum32())
}
