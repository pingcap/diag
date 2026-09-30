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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pingcap/diag/collector/log/parser"
	"github.com/stretchr/testify/require"
)

// rocksdbJSONFixture is the same log written through TiKV's JSON formatter,
// which is used for every drain - rocksdb included - when log.format = "json".
// The keys are the ones TiKV emits: time, level, caller, message.
const rocksdbJSONFixture = `{"time":"2024/01/01 09:59:59.000 +08:00","level":"INFO","caller":"db/version_set.cc:1","message":"RocksDB version: 8.11.3"}
{"time":"2024/01/01 10:00:00.000 +08:00","level":"INFO","caller":"db/x.cc:1","message":"first message"}
{"time":"2024/01/01 10:30:00.000 +08:00","level":"WARN","caller":"db/x.cc:1","message":"second message"}
{"time":"2024/01/01 12:00:00.000 +08:00","level":"INFO","caller":"db/x.cc:1","message":"fourth message"}
`

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	fi, err := os.Stat(path)
	require.NoError(t, err)
	return fi
}

// A rocksdb log as TiKV writes it. Every line is stamped, including the RocksDB
// version and options block: RocksDB emits that block through
// Logger::LogHeader, whose default implementation logs it at INFO level, so it
// is neither tagged as a header nor written without a timestamp. Only the
// continuation lines of a record that spans several physical lines have no
// timestamp of their own.
const rocksdbFixture = `[2024/01/01 09:59:59.000 +08:00][5][INFO] RocksDB version: 8.11.3
[2024/01/01 09:59:59.000 +08:00][5][INFO] Options.error_if_exists: 0
[2024/01/01 10:00:00.000 +08:00][12345][INFO] first message
[2024/01/01 10:30:00.000 +08:00][12345][INFO] second message
[2024/01/01 11:00:00.000 +08:00][12345][WARN] third message
  continuation of the third message
[2024/01/01 12:00:00.000 +08:00][12345][INFO] fourth message
`

// rocksdbUnstamped stands for a log whose lines carry no timestamp at all, as
// produced when log.enable-timestamp is disabled.
const rocksdbUnstamped = `RocksDB version: 8.11.3
Options.error_if_exists: 0
12345 [db/version_set.cc:1] no timestamp here
`

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(parser.TimeStampLayout, s)
	require.NoError(t, err)
	return v
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	fp := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(fp, []byte(content), 0o644))
	return fp
}

func TestRocksDBLineTime(t *testing.T) {
	assert := require.New(t)

	cases := []struct {
		line string
		want string
		ok   bool
	}{
		{"[2024/01/01 12:00:00.000 +08:00][12345][INFO] hello", "2024/01/01 12:00:00.000 +08:00", true},
		{"[2024/01/01 12:00:00.000 -05:00][1][WARN] hello", "2024/01/01 12:00:00.000 -05:00", true},
		// the RocksDB version line is stamped like any other line
		{"[2024/01/01 12:00:00.000 +08:00][5][INFO] RocksDB version: 8.10.2", "2024/01/01 12:00:00.000 +08:00", true},
		// a continuation line of a multi-line record
		{"  continuation of the previous message", "", false},
		// only the leading timestamp matters, the shape of the rest of the line
		// is not checked: the extractor is applied to files that the file name
		// already classified as rocksdb logs
		{"[2024/01/01 12:00:00.000 +08:00] [INFO] [mod.rs:1] [\"msg\"]", "2024/01/01 12:00:00.000 +08:00", true},
	}

	for _, cs := range cases {
		got, ok := rocksDBLineTime([]byte(cs.line))
		assert.Equal(cs.ok, ok, cs.line)
		if cs.ok {
			assert.Equal(cs.want, got.Format(parser.TimeStampLayout), cs.line)
		}
	}
}

func TestStdAndSlowLineTime(t *testing.T) {
	assert := require.New(t)

	// a component log line
	got, ok := stdLineTime([]byte(`[2024/01/01 12:00:00.000 +08:00] [INFO] [mod.rs:1] ["msg"]`))
	assert.True(ok)
	assert.Equal("2024/01/01 12:00:00.000 +08:00", got.Format(parser.TimeStampLayout))

	// a continuation line has no timestamp of its own
	_, ok = stdLineTime([]byte("  at main.go:12"))
	assert.False(ok)

	// a slow query log header
	got, ok = slowLineTime([]byte("# Time: 2024-01-01T12:00:00.000000+08:00"))
	assert.True(ok)
	assert.Equal(int64(1704081600), got.Unix())

	assert.Nil(lineTimeOf(LogTypeUnknown), "unknown logs cannot be trimmed line by line")
}

func TestFileHeadInRange(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()
	fp := writeFile(t, dir, "rocksdb.info", rocksdbFixture)
	fi, err := os.Stat(fp)
	assert.NoError(err)

	// the file overlaps the range
	assert.True(fileHeadInRange(fp, fi, rocksDBLineTime,
		mustTime(t, "2024/01/01 09:00:00.000 +08:00"),
		mustTime(t, "2024/01/01 10:30:00.000 +08:00")))

	// the file starts after the end of the range, so it is newer than what is
	// asked for
	assert.False(fileHeadInRange(fp, fi, rocksDBLineTime,
		mustTime(t, "2024/01/01 08:00:00.000 +08:00"),
		mustTime(t, "2024/01/01 09:00:00.000 +08:00")))

	// nothing was appended after the range began
	old := mustTime(t, "2023/12/31 23:00:00.000 +08:00")
	assert.NoError(os.Chtimes(fp, old, old))
	fi, err = os.Stat(fp)
	assert.NoError(err)
	assert.False(fileHeadInRange(fp, fi, rocksDBLineTime,
		mustTime(t, "2024/01/01 09:00:00.000 +08:00"),
		mustTime(t, "2024/01/01 10:00:00.000 +08:00")))

	// a file without any timestamp is kept instead of being dropped
	unstamped := writeFile(t, dir, "rocksdb-unstamped.info", rocksdbUnstamped)
	fi, err = os.Stat(unstamped)
	assert.NoError(err)
	assert.True(fileHeadInRange(unstamped, fi, rocksDBLineTime,
		mustTime(t, "2024/01/01 09:00:00.000 +08:00"),
		mustTime(t, "2024/01/01 10:00:00.000 +08:00")))
}

func TestTrimToRange(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()
	src := writeFile(t, dir, "rocksdb.info", rocksdbFixture)
	dst := filepath.Join(dir, "out", "rocksdb.info")

	// out of range lines are dropped, the continuation line of a kept record
	// survives with it
	written, found, err := trimToRange(src, dst, rocksDBLineTime,
		mustTime(t, "2024/01/01 10:15:00.000 +08:00"),
		mustTime(t, "2024/01/01 11:30:00.000 +08:00"))
	assert.NoError(err)
	assert.True(found)
	expected := `[2024/01/01 10:30:00.000 +08:00][12345][INFO] second message
[2024/01/01 11:00:00.000 +08:00][12345][WARN] third message
  continuation of the third message
`
	assert.Equal(int64(len(expected)), written)
	content, err := os.ReadFile(dst)
	assert.NoError(err)
	assert.Equal(expected, string(content))

	// a dropped record takes its continuation lines with it
	_, _, err = trimToRange(src, dst, rocksDBLineTime,
		mustTime(t, "2024/01/01 10:30:00.000 +08:00"),
		mustTime(t, "2024/01/01 10:45:00.000 +08:00"))
	assert.NoError(err)
	content, err = os.ReadFile(dst)
	assert.NoError(err)
	assert.Equal("[2024/01/01 10:30:00.000 +08:00][12345][INFO] second message\n", string(content))

	// a line without any timestamp is reported as "not trimmable"
	_, found, err = trimToRange(writeFile(t, dir, "unstamped.info", rocksdbUnstamped), dst,
		rocksDBLineTime,
		mustTime(t, "2024/01/01 10:00:00.000 +08:00"),
		mustTime(t, "2024/01/01 11:00:00.000 +08:00"))
	assert.NoError(err)
	assert.False(found)

	// an empty file is not trimmable either
	_, found, err = trimToRange(writeFile(t, dir, "empty.info", ""), dst,
		rocksDBLineTime,
		mustTime(t, "2024/01/01 10:00:00.000 +08:00"),
		mustTime(t, "2024/01/01 11:00:00.000 +08:00"))
	assert.NoError(err)
	assert.False(found)
}

func TestTrimToRangeKeepsLeadingUnstampedLine(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()

	// A file may start with an unstamped line, e.g. a record split by a
	// rotation. It has nothing to inherit from, so it is kept.
	src := writeFile(t, dir, "split.info",
		"  tail of a record split by a rotation\n"+
			"[2024/01/01 12:00:00.000 +08:00][1][INFO] out of range\n")
	dst := filepath.Join(dir, "out", "split.info")

	_, found, err := trimToRange(src, dst, rocksDBLineTime,
		mustTime(t, "2024/01/01 10:00:00.000 +08:00"),
		mustTime(t, "2024/01/01 11:00:00.000 +08:00"))
	assert.NoError(err)
	assert.True(found)

	content, err := os.ReadFile(dst)
	assert.NoError(err)
	assert.Equal("  tail of a record split by a rotation\n", string(content))
}

func TestTrimmedPath(t *testing.T) {
	assert := require.New(t)
	trim := t.TempDir()

	// the whole source path is kept, so two instances of the same host do not
	// collide on their rocksdb.info
	a := trimmedPath(trim, "/data1/tidb-data/tikv-20160/rocksdb.info")
	b := trimmedPath(trim, "/data1/tidb-data/tikv-20161/rocksdb.info")
	assert.NotEqual(a, b)
	assert.Equal(filepath.Join(trim, "data1/tidb-data/tikv-20160/rocksdb.info"), a)
	assert.Equal(filepath.Join(trim, "data1/tidb-data/tikv-20161/rocksdb.info"), b)

	// a relative source path stays inside the trim directory
	rel := trimmedPath(trim, "../../etc/passwd")
	assert.True(strings.HasPrefix(rel, trim+string(filepath.Separator)), rel)
	assert.NotContains(rel, "..")
}

func TestLogScraperTrim(t *testing.T) {
	assert := require.New(t)
	dataDir := t.TempDir()
	trimDir := t.TempDir()

	active := writeFile(t, dataDir, "rocksdb.info", rocksdbFixture)
	// a rotated log from before the range: it must not be downloaded at all
	rotated := writeFile(t, dataDir, "rocksdb-2023-12-30T00-00-00.000.info",
		"[2023/12/30 00:00:00.000 +08:00][1][INFO] ancient\n")
	old := mustTime(t, "2023/12/30 01:00:00.000 +08:00")
	assert.NoError(os.Chtimes(rotated, old, old))
	// an unrelated file of the data directory must not be collected
	writeFile(t, dataDir, "000123.sst", "\x00\x01\x02")

	start := mustTime(t, "2024/01/01 10:15:00.000 +08:00")
	end := mustTime(t, "2024/01/01 11:30:00.000 +08:00")

	s := &LogScraper{
		Paths:   []string{filepath.Join(dataDir, "*")},
		Types:   map[string]bool{LogTypeRocksDB: true},
		Start:   start,
		End:     end,
		Trim:    true,
		TrimDir: trimDir,
	}
	sample := &Sample{}
	assert.NoError(s.Scrap(sample))

	assert.Len(sample.Log, 1, "only the active rocksdb log is in range")
	target := trimmedPath(trimDir, active)
	expected := `[2024/01/01 10:30:00.000 +08:00][12345][INFO] second message
[2024/01/01 11:00:00.000 +08:00][12345][WARN] third message
  continuation of the third message
`
	assert.Equal(int64(len(expected)), sample.Log[target])
	assert.Equal(LogTypeRocksDB, sample.LogTypes[target])

	content, err := os.ReadFile(target)
	assert.NoError(err)
	assert.Equal(expected, string(content))
	assert.NotContains(string(content), "first message")
	assert.NotContains(string(content), "fourth message")

	// without trimming the whole file is reported
	s.Trim = false
	sample = &Sample{}
	assert.NoError(s.Scrap(sample))
	assert.Len(sample.Log, 1)
	assert.Equal(int64(len(rocksdbFixture)), sample.Log[active])

	// trimming requires a directory to write to
	s.Trim = true
	s.TrimDir = ""
	assert.Error(s.Scrap(&Sample{}))
}

func TestLogScraperTrimFallsBackToWholeFile(t *testing.T) {
	assert := require.New(t)
	dataDir := t.TempDir()
	trimDir := t.TempDir()

	// log.enable-timestamp = false makes TiKV write rocksdb lines without a
	// timestamp. Nothing can be trimmed, the file has to be collected as is.
	fp := writeFile(t, dataDir, "rocksdb.info", rocksdbUnstamped)

	s := &LogScraper{
		Paths:   []string{filepath.Join(dataDir, "*")},
		Types:   map[string]bool{LogTypeRocksDB: true},
		Start:   mustTime(t, "2024/01/01 10:00:00.000 +08:00"),
		End:     mustTime(t, "2024/01/01 11:00:00.000 +08:00"),
		Trim:    true,
		TrimDir: trimDir,
	}
	sample := &Sample{}
	assert.NoError(s.Scrap(sample))

	assert.Len(sample.Log, 1)
	assert.Equal(int64(len(rocksdbUnstamped)), sample.Log[fp],
		"the original file is reported when nothing can be trimmed")
	_, err := os.Stat(trimmedPath(trimDir, fp))
	assert.True(os.IsNotExist(err), "no trimmed copy must be left behind")
}

// ---------------------------------------------------------------------------
// every format TiKV can write
// ---------------------------------------------------------------------------

// TestRocksDBTrimsEveryLogFormat is the guard for the whole class of "the file
// is recognised by its name, but its content is not what we assumed" bugs: a
// format that is not understood makes the trimming silently degrade to
// collecting the file in full, which is exactly the defect this feature fixes.
// A new format needs a row here.
func TestRocksDBTrimsEveryLogFormat(t *testing.T) {
	assert := require.New(t)
	start := mustTime(t, "2024/01/01 10:15:00.000 +08:00")
	end := mustTime(t, "2024/01/01 11:00:00.000 +08:00")

	cases := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:    "text",
			content: rocksdbFixture,
			expected: `[2024/01/01 10:30:00.000 +08:00][12345][INFO] second message
[2024/01/01 11:00:00.000 +08:00][12345][WARN] third message
  continuation of the third message
`,
		},
		{
			name:    "json",
			content: rocksdbJSONFixture,
			expected: `{"time":"2024/01/01 10:30:00.000 +08:00","level":"WARN","caller":"db/x.cc:1","message":"second message"}
`,
		},
	}

	for _, cs := range cases {
		dir := t.TempDir()
		src := writeFile(t, dir, "rocksdb.info", cs.content)
		dst := filepath.Join(dir, "out", "rocksdb.info")

		// line level: only the records inside the window survive
		written, found, err := trimToRange(src, dst, rocksDBLineTime, start, end)
		assert.NoError(err, cs.name)
		assert.True(found, "%s: no timestamp recognised, trimming would silently be skipped", cs.name)
		content, err := os.ReadFile(dst)
		assert.NoError(err, cs.name)
		assert.Equal(cs.expected, string(content), "%s: trimmed content", cs.name)
		assert.Equal(int64(len(cs.expected)), written, cs.name)

		// file level: a file that lies entirely after the window is dropped, so
		// rotated logs outside the range are not downloaded at all
		assert.False(fileHeadInRange(src, mustStat(t, src), rocksDBLineTime,
			mustTime(t, "2024/01/01 08:00:00.000 +08:00"),
			mustTime(t, "2024/01/01 09:00:00.000 +08:00")),
			"%s: a file newer than the window must be excluded", cs.name)

		assert.True(fileHeadInRange(src, mustStat(t, src), rocksDBLineTime,
			mustTime(t, "2024/01/01 10:00:00.000 +08:00"),
			mustTime(t, "2024/01/01 11:00:00.000 +08:00")),
			"%s: a file overlapping the window must be kept", cs.name)
	}
}

// TestRocksDBLineTimeRejections covers the shapes the extractor has to refuse:
// a line it can not read a time from must not be mistaken for a record, and a
// file full of them is collected in full rather than emptied.
func TestRocksDBLineTimeRejections(t *testing.T) {
	assert := require.New(t)

	rejected := []string{
		`{"level":"INFO","caller":"x:1","message":"no time field"}`,
		`{"time":"not a timestamp","level":"INFO","message":"x"}`,
		`{"time":"2024/01/01 10:00:00.000 +08:00","message":"no level"}`,
		`{"time":123,"level":"INFO"}`,
		`{"time":"2024/01/01 10:00:00.000 +08:00"`,
		`[2024/13/45 99:99:99.999 +08:00][5][INFO] impossible time`,
		`  continuation line`,
	}
	for _, line := range rejected {
		_, ok := rocksDBLineTime([]byte(line))
		assert.False(ok, "must not yield a timestamp: %s", line)
	}

	// TiKV can also be configured without timestamps: valid JSON without a time
	// field means the file has to be collected whole
	dir := t.TempDir()
	src := writeFile(t, dir, "rocksdb.info",
		`{"level":"INFO","caller":"x:1","message":"no time"}`+"\n")
	dst := filepath.Join(dir, "out", "rocksdb.info")
	_, found, err := trimToRange(src, dst, rocksDBLineTime,
		mustTime(t, "2024/01/01 10:00:00.000 +08:00"), mustTime(t, "2024/01/01 11:00:00.000 +08:00"))
	assert.NoError(err)
	assert.False(found, "a file without timestamps must be reported as not trimmable")
}

// TestFileHeadInRangeHeadScanLimit pins the look ahead: TiKV stamps every line,
// so the first one normally decides, but a file whose first timestamp sits
// deeper than the scan window has to be kept rather than silently dropped.
func TestFileHeadInRangeHeadScanLimit(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()
	start := mustTime(t, "2024/01/01 10:00:00.000 +08:00")
	end := mustTime(t, "2024/01/01 11:00:00.000 +08:00")

	build := func(leading int) string {
		var b strings.Builder
		for i := 0; i < leading; i++ {
			b.WriteString("  continuation without a timestamp\n")
		}
		b.WriteString("[2024/01/01 12:00:00.000 +08:00][5][INFO] after the range\n")
		return b.String()
	}

	near := writeFile(t, dir, "near.info", build(maxLeadingLines-1))
	assert.False(fileHeadInRange(near, mustStat(t, near), rocksDBLineTime, start, end),
		"a timestamp inside the scan window decides")

	far := writeFile(t, dir, "far.info", build(maxLeadingLines+1))
	assert.True(fileHeadInRange(far, mustStat(t, far), rocksDBLineTime, start, end),
		"a file whose timestamps are all beyond the scan window is kept, never dropped")
}

func TestTrimToRangeSourceOpenFailure(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()
	_, _, err := trimToRange(filepath.Join(dir, "missing.info"), filepath.Join(dir, "out"),
		rocksDBLineTime,
		mustTime(t, "2024/01/01 10:00:00.000 +08:00"), mustTime(t, "2024/01/01 11:00:00.000 +08:00"))
	assert.Error(err, "an unreadable source has to surface as an error, not as an empty copy")
}

// TestTrimFallsBackWhenTheDestinationCannotBeCreated exercises the failure after
// the directory is in place: the copy can not be created at all, so the original
// file must be reported and no partial copy left behind.
func TestTrimFallsBackWhenTheDestinationCannotBeCreated(t *testing.T) {
	assert := require.New(t)
	dir, trimDir := t.TempDir(), t.TempDir()
	src := writeFile(t, dir, "rocksdb.info", rocksdbFixture)

	// a directory exactly where the trimmed copy would be written
	dst := trimmedPath(trimDir, src)
	assert.NoError(os.MkdirAll(dst, 0o755))

	s := &LogScraper{
		Paths:   []string{src},
		Types:   map[string]bool{LogTypeRocksDB: true},
		Start:   mustTime(t, "2024/01/01 10:15:00.000 +08:00"),
		End:     mustTime(t, "2024/01/01 11:00:00.000 +08:00"),
		Trim:    true,
		TrimDir: trimDir,
	}
	sample := &Sample{}
	assert.NoError(s.Scrap(sample))
	assert.Equal(int64(len(rocksdbFixture)), sample.Log[src])
	_, err := os.Stat(filepath.Join(dst, "rocksdb.info"))
	assert.True(os.IsNotExist(err), "no partial copy may be reported")
}

// TestTrimStdAndSlowRecords walks the other two log types through the trimming.
// Both carry their time in a different shape and both have multi line records,
// which have to follow the record they belong to.
func TestTrimStdAndSlowRecords(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()
	start := mustTime(t, "2024/01/01 10:15:00.000 +08:00")
	end := mustTime(t, "2024/01/01 11:00:00.000 +08:00")

	trim := func(name, logtype, content string) string {
		t.Helper()
		src := writeFile(t, dir, name, content)
		dst := filepath.Join(dir, "out", name)
		_, found, err := trimToRange(src, dst, lineTimeOf(logtype), start, end)
		assert.NoError(err, name)
		assert.True(found, name)
		out, err := os.ReadFile(dst)
		assert.NoError(err, name)
		return string(out)
	}

	// a component log record with a stack trace attached
	stdLog := `[2024/01/01 10:00:00.000 +08:00] [INFO] [mod.rs:1] ["before"]
[2024/01/01 10:30:00.000 +08:00] [ERROR] [mod.rs:2] ["inside"]
  at foo.rs:10
  at bar.rs:20
[2024/01/01 12:00:00.000 +08:00] [INFO] [mod.rs:3] ["after"]
`
	assert.Equal(`[2024/01/01 10:30:00.000 +08:00] [ERROR] [mod.rs:2] ["inside"]
  at foo.rs:10
  at bar.rs:20
`, trim("tikv.log", LogTypeStd, stdLog))

	// a slow query record: only the head line carries the time, the body has to
	// follow it
	slowLog := `# Time: 2024-01-01T10:00:00.000000+08:00
# User: u
select 1;
# Time: 2024-01-01T10:30:00.000000+08:00
# User: u
select 2;
# Time: 2024-01-01T12:00:00.000000+08:00
# User: u
select 3;
`
	assert.Equal(`# Time: 2024-01-01T10:30:00.000000+08:00
# User: u
select 2;
`, trim("tidb_slow_query.log", LogTypeSlow, slowLog))
}

func TestLineTimeOfDispatchesEachType(t *testing.T) {
	assert := require.New(t)

	cases := []struct {
		logtype string
		line    string
	}{
		{LogTypeRocksDB, "[2024/01/01 10:00:00.000 +08:00][5][INFO] rocksdb"},
		{LogTypeRocksDB, `{"time":"2024/01/01 10:00:00.000 +08:00","level":"INFO","caller":"x:1","message":"rocksdb"}`},
		{LogTypeStd, `[2024/01/01 10:00:00.000 +08:00] [INFO] [mod.rs:1] ["msg"]`},
		{LogTypeSlow, "# Time: 2024-01-01T10:00:00.000000+08:00"},
	}
	for _, cs := range cases {
		extract := lineTimeOf(cs.logtype)
		assert.NotNil(extract, cs.logtype)
		got, ok := extract([]byte(cs.line))
		assert.True(ok, "%s: %s", cs.logtype, cs.line)
		assert.Equal("2024/01/01 10:00:00.000 +08:00", got.Format(parser.TimeStampLayout), cs.logtype)
	}

	assert.Nil(lineTimeOf(LogTypeUnknown), "unknown logs carry no per line timestamp")
}

// ---------------------------------------------------------------------------
// the invariant, instead of a handful of examples
// ---------------------------------------------------------------------------

// TestTrimToRangeKeepsExactlyTheRecordsInRange is a property test: it builds a
// log of multi-line records at known times and checks, for every kind of
// window, that the output holds exactly the records inside it together with
// their continuation lines - no boundary, continuation or last-line rule can
// break without failing here.
func TestTrimToRangeKeepsExactlyTheRecordsInRange(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()

	type record struct {
		minute int
		lines  []string
	}
	records := make([]record, 0, 12)
	for i := 0; i < 12; i++ {
		lines := []string{fmt.Sprintf("[2024/01/01 00:%02d:00.000 +08:00][%d][INFO] record %d", i, i, i)}
		for c := 0; c <= i%3; c++ { // 0 to 2 continuation lines
			lines = append(lines, fmt.Sprintf("  continuation %d of record %d", c, i))
		}
		records = append(records, record{minute: i, lines: lines})
	}

	var whole strings.Builder
	for _, r := range records {
		for _, l := range r.lines {
			whole.WriteString(l + "\n")
		}
	}
	src := writeFile(t, dir, "rocksdb.info", whole.String())

	between := func(lo, hi int) string {
		var b strings.Builder
		for _, r := range records {
			if r.minute < lo || r.minute > hi {
				continue
			}
			for _, l := range r.lines {
				b.WriteString(l + "\n")
			}
		}
		return b.String()
	}

	windows := [][2]int{
		{0, 0}, {5, 5}, {2, 7}, {0, 11}, {1, 3}, {9, 11},
		{20, 30}, // nothing in range at all
		{7, 4},   // start after end
	}
	for _, w := range windows {
		dst := filepath.Join(dir, fmt.Sprintf("out-%d-%d", w[0], w[1]), "rocksdb.info")
		start := mustTime(t, fmt.Sprintf("2024/01/01 00:%02d:00.000 +08:00", w[0]))
		end := mustTime(t, fmt.Sprintf("2024/01/01 00:%02d:00.000 +08:00", w[1]))

		written, found, err := trimToRange(src, dst, rocksDBLineTime, start, end)
		assert.NoError(err, "window %v", w)
		assert.True(found, "window %v", w)

		content, err := os.ReadFile(dst)
		assert.NoError(err, "window %v", w)
		assert.Equal(between(w[0], w[1]), string(content), "window %v", w)
		assert.Equal(int64(len(between(w[0], w[1]))), written, "window %v", w)
	}
}

// TestTrimToRangeLineBoundaries guards the byte level rules: a record must not
// disappear because of how the file ends, which line separator it uses, or how
// long a single line is.
func TestTrimToRangeLineBoundaries(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()
	start := mustTime(t, "2024/01/01 10:15:00.000 +08:00")
	end := mustTime(t, "2024/01/01 11:00:00.000 +08:00")

	trim := func(name, content string) string {
		t.Helper()
		src := writeFile(t, dir, name, content)
		dst := filepath.Join(dir, "out", name)
		_, found, err := trimToRange(src, dst, rocksDBLineTime, start, end)
		assert.NoError(err, name)
		assert.True(found, name)
		out, err := os.ReadFile(dst)
		assert.NoError(err, name)
		return string(out)
	}

	// a missing final newline must not cost the last record
	assert.Equal("[2024/01/01 10:30:00.000 +08:00][1][INFO] last",
		trim("nonewline.info", "[2024/01/01 10:30:00.000 +08:00][1][INFO] last"))

	// CRLF is kept byte for byte
	assert.Equal("[2024/01/01 10:30:00.000 +08:00][1][INFO] a\r\n",
		trim("crlf.info",
			"[2024/01/01 10:30:00.000 +08:00][1][INFO] a\r\n"+
				"[2024/01/01 12:00:00.000 +08:00][1][INFO] b\r\n"))

	// a line far longer than any internal buffer is neither split nor dropped
	long := "[2024/01/01 10:30:00.000 +08:00][1][INFO] " + strings.Repeat("x", 512*1024) + "\n"
	assert.Equal(long, trim("long.info", long+
		"[2024/01/01 12:00:00.000 +08:00][1][INFO] after\n"))
}

// ---------------------------------------------------------------------------
// never lose data: the fallbacks
// ---------------------------------------------------------------------------

// TestTrimFallsBackWhenTheCopyCannotBeWritten: if the trimmed copy can not be
// produced the original file has to be collected, never an empty one and never
// an error that aborts the collection.
func TestTrimFallsBackWhenTheCopyCannotBeWritten(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()
	src := writeFile(t, dir, "rocksdb.info", rocksdbFixture)
	// a regular file where the trim directory should be, so nothing can be
	// created below it
	blocker := writeFile(t, dir, "not-a-dir", "")

	s := &LogScraper{
		Paths:   []string{filepath.Join(dir, "rocksdb.info")},
		Types:   map[string]bool{LogTypeRocksDB: true},
		Start:   mustTime(t, "2024/01/01 10:15:00.000 +08:00"),
		End:     mustTime(t, "2024/01/01 11:00:00.000 +08:00"),
		Trim:    true,
		TrimDir: blocker,
	}
	sample := &Sample{}
	assert.NoError(s.Scrap(sample))

	assert.Len(sample.Log, 1)
	assert.Equal(int64(len(rocksdbFixture)), sample.Log[src],
		"the original file must be collected when the trimmed copy fails")
}

// TestTrimIsANoOpForUntrimmableTypes: a type without per line timestamps can
// not be trimmed, and --trim must then behave as if it were not given.
func TestTrimIsANoOpForUntrimmableTypes(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()
	trimDir := t.TempDir()
	content := "no timestamp here\nanother line\n"
	src := writeFile(t, dir, "mystery.dat", content)

	s := &LogScraper{
		Paths:   []string{filepath.Join(dir, "*")},
		Types:   map[string]bool{LogTypeUnknown: true},
		Start:   mustTime(t, "2024/01/01 10:00:00.000 +08:00"),
		End:     mustTime(t, "2024/01/01 11:00:00.000 +08:00"),
		Trim:    true,
		TrimDir: trimDir,
	}
	sample := &Sample{}
	assert.NoError(s.Scrap(sample))

	assert.Len(sample.Log, 1)
	assert.Equal(int64(len(content)), sample.Log[src])
	entries, err := os.ReadDir(trimDir)
	assert.NoError(err)
	assert.Empty(entries, "nothing may be written for a type that cannot be trimmed")
}

// TestLogScraperSkipsDirectories: the glob used in production is
// "<data dir>/*", and a TiKV data directory contains sub directories such as
// db, raft and snap.
func TestLogScraperSkipsDirectories(t *testing.T) {
	assert := require.New(t)
	dataDir := t.TempDir()
	trimDir := t.TempDir()
	assert.NoError(os.MkdirAll(filepath.Join(dataDir, "db"), 0o755))
	assert.NoError(os.MkdirAll(filepath.Join(dataDir, "raft"), 0o755))
	src := writeFile(t, dataDir, "rocksdb.info", rocksdbFixture)

	s := &LogScraper{
		Paths:   []string{filepath.Join(dataDir, "*")},
		Types:   map[string]bool{LogTypeRocksDB: true},
		Start:   mustTime(t, "2024/01/01 10:15:00.000 +08:00"),
		End:     mustTime(t, "2024/01/01 11:00:00.000 +08:00"),
		Trim:    true,
		TrimDir: trimDir,
	}
	sample := &Sample{}
	assert.NoError(s.Scrap(sample))

	assert.Len(sample.Log, 1, "only the file, never a directory")
	_, ok := sample.Log[trimmedPath(trimDir, src)]
	assert.True(ok)
}
