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
	"io"
	"io/fs"
	"os"
	"path/filepath"
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
	Paths     []string        // paths of log files
	Types     map[string]bool // log type
	Start     time.Time       // start time
	End       time.Time       // end time
	outputDir string
}

// Scrap implements the Scraper interface
func (s *LogScraper) Scrap(result *Sample) error {
	if result.Log == nil {
		result.Log = make(FileStat)
	}
	if result.LogTypes == nil {
		result.LogTypes = make(FileTypes)
	}
	if result.LogTargets == nil {
		result.LogTargets = make(FileTypes)
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
		if fi, err := os.Stat(fp); err == nil {
			if fi.IsDir() {
				continue
			}

			logtype, in, err := getLogType(fp, fi, s.Start, s.End)
			if s.Types[logtype] && in {
				target := fp
				size := fi.Size()
				if canFilterLogFile(fp, logtype) {
					filtered, filteredSize, ok, err := s.filterLogFile(fp, logtype)
					if err != nil {
						fmt.Fprintf(os.Stderr, "error filtering %s: %s\n", fi.Name(), err)
						continue
					}
					if !ok {
						continue
					}
					target = filtered
					size = filteredSize
				}
				result.Log[target] = size
				result.LogTargets[target] = fp
				result.LogTypes[target] = logtype
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "error checking %s: %s\n", fi.Name(), err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "error checking %s: %s\n", fi.Name(), err)
		}
	}

	return nil
}

func getLogType(fpath string, fi fs.FileInfo, start, end time.Time) (logtype string, inrange bool, err error) {
	fileName := filepath.Base(fpath)
	// collect stderr log despite time range
	if strings.Contains(fileName, "stderr") {
		return LogTypeStd, true, nil
	}

	// todo: parse time range
	if strings.HasPrefix(fileName, "rocksdb") && strings.HasSuffix(fileName, ".info") {
		return LogTypeRocksDB, true, nil
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

func canFilterLogFile(fpath, logtype string) bool {
	if strings.Contains(filepath.Base(fpath), "stderr") {
		return false
	}
	return logtype == LogTypeStd || logtype == LogTypeSlow
}

func (s *LogScraper) filterLogFile(fpath, logtype string) (string, int64, bool, error) {
	outputDir, err := s.filteredOutputDir()
	if err != nil {
		return "", 0, false, err
	}
	outPath := filteredLogPath(outputDir, fpath)
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return "", 0, false, err
	}

	in, err := os.Open(fpath)
	if err != nil {
		return "", 0, false, err
	}
	defer in.Close()

	var reader io.Reader = in
	var gzReader *gzip.Reader
	if strings.HasSuffix(fpath, ".gz") {
		gzReader, err = gzip.NewReader(in)
		if err != nil {
			return "", 0, false, err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	out, err := os.Create(outPath)
	if err != nil {
		return "", 0, false, err
	}
	var gzWriter *gzip.Writer
	defer func() {
		if gzWriter != nil {
			_ = gzWriter.Close()
		}
		_ = out.Close()
	}()

	var writer io.Writer = out
	if strings.HasSuffix(outPath, ".gz") {
		gzWriter = gzip.NewWriter(out)
		writer = gzWriter
	}

	written, err := writeFilteredLog(reader, writer, logtype, s.Start, s.End)
	if err != nil {
		return "", 0, false, err
	}
	if gzWriter != nil {
		if err := gzWriter.Close(); err != nil {
			return "", 0, false, err
		}
		gzWriter = nil
	}
	if err := out.Close(); err != nil {
		return "", 0, false, err
	}
	if !written {
		_ = os.Remove(outPath)
		return "", 0, false, nil
	}
	fi, err := os.Stat(outPath)
	if err != nil {
		return "", 0, false, err
	}
	return outPath, fi.Size(), true, nil
}

func (s *LogScraper) filteredOutputDir() (string, error) {
	if s.outputDir != "" {
		return s.outputDir, nil
	}
	outputDir, err := os.MkdirTemp("", "diag-scraped-logs-*")
	if err != nil {
		return "", err
	}
	s.outputDir = outputDir
	return s.outputDir, nil
}

func filteredLogPath(outputDir, src string) string {
	clean := filepath.Clean(src)
	if filepath.IsAbs(clean) {
		clean = strings.TrimPrefix(clean, string(filepath.Separator))
	}
	return filepath.Join(outputDir, clean)
}

func writeFilteredLog(reader io.Reader, writer io.Writer, logtype string, start, end time.Time) (bool, error) {
	bufr := bufio.NewReader(reader)
	parsers := parser.ListStd()
	if logtype == LogTypeSlow {
		parsers = []parser.Parser{&parser.SlowQueryParser{}}
	}

	var current []byte
	currentInRange := false
	written := false
	flush := func() error {
		if currentInRange && len(current) > 0 {
			if _, err := writer.Write(current); err != nil {
				return err
			}
			written = true
		}
		current = nil
		currentInRange = false
		return nil
	}

	for {
		line, err := bufr.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return false, err
		}
		if len(line) > 0 {
			if ts := parseLine(bytes.TrimRight(line, "\r\n"), parsers); ts != nil {
				if err := flush(); err != nil {
					return false, err
				}
				if ts.After(end) {
					break
				}
				current = append(current, line...)
				currentInRange = !ts.Before(start)
			} else if current != nil {
				current = append(current, line...)
			}
		}
		if err == io.EOF {
			break
		}
	}
	if err := flush(); err != nil {
		return false, err
	}
	return written, nil
}

func parseLine(line []byte, parsers []parser.Parser) *time.Time {
	for _, p := range parsers {
		if t, _ := p.ParseHead(line); t != nil {
			return t
		}
	}
	return nil
}
