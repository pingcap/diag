package searcher_test

import (
	"bufio"
	"io"
	"strings"
	"testing"
	"time"

	. "github.com/pingcap/check"
	"github.com/pingcap/tidb-foresight/searcher"
)

func TestAdaptor(t *testing.T) {
	TestingT(t)
}

type AdaptorTestSuite struct{}

var _ = Suite(&AdaptorTestSuite{})

var TiDBLogExample = []string{
	`[2019/07/19 15:15:35.846 +08:00] [INFO] [gc_worker.go:432] ["[gc worker] start redo-delete ranges"] [uuid=5aec892a9a40002] ["num of ranges"=0]`,
	`[2019/07/19 15:15:wrong_text35.846 +08:00] [INFO] [gc_worker.go:451] ["[gc worker] finish redo-delete ranges"] [uuid=5aec892a9a40002] ["num of ranges"=0] ["cost time"=441ns]`,
	`[2019/07/19 15:16:35.814 +08:00] [INFO] [gc_worker.go:226] ["[gc worker] there's already a gc job running, skipped"]`,
}

func (a *AdaptorTestSuite) TestTiDBParser(c *C) {
	searchStr := "redo"
	logStr := strings.Join(TiDBLogExample, "\n")
	reader := bufio.NewReader(strings.NewReader(logStr))
	parser := searcher.Tidbsearcher{
		BaseParser: searcher.BaseParser{
			Reader:      reader,
			SearchBytes: []byte(searchStr),
		},
	}
	t := time.Unix(1563520535, 846000000)
	expect := &searcher.Item{
		Time:  &t,
		Level: searcher.LevelINFO,
		Line:  TiDBLogExample[0],
		Type:  searcher.TypeTiDB,
	}

	err := parser.Next()
	c.Assert(err, IsNil)
	item := parser.GetCurrentLog()
	c.Assert(item, DeepEquals, expect)

	err = parser.Next()
	c.Assert(err, NotNil)
	item = parser.GetCurrentLog()
	c.Assert(item, IsNil)

	err = parser.Next()
	c.Assert(err, Equals, io.EOF)
	item = parser.GetCurrentLog()
	c.Assert(item, IsNil)
}

var TiKVLogExample = []string{
	`2019/07/18 23:22:38.182 INFO apply.rs:910: [region 12] 13 execute admin command cmd_type: CompactLog compact_log {compact_index: 57295 compact_term: 6} at [term: 6, index: 57297]`,
	`2019/07/18 23:43:45.481 INFO apply.rs:910: [region 12] 13 execute admin command cmd_type: CompactLog compact_log {compact_index: 57346 compact_term: 6} at [term: 6, index: 57348]`,
	`2019/07/19 00:04:43.582 INFO apply.rs:910: [region 12] 13 execute admin command cmd_type: CompactLog compact_log {compact_index: 57397 compact_term: 6} at [term: 6, index: 57399]`,
}

func (a *AdaptorTestSuite) TestTiKVParser(c *C) {
	searchStr := "57346"
	logStr := strings.Join(TiKVLogExample, "\n")
	reader := bufio.NewReader(strings.NewReader(logStr))
	parser := searcher.Tikvsearcher{
		BaseParser: searcher.BaseParser{
			Reader:      reader,
			SearchBytes: []byte(searchStr),
		},
	}
	loc, err := time.LoadLocation("Asia/Chongqing")
	c.Assert(err, IsNil)

	t := time.Unix(1563464625, 481000000).In(loc)
	expect := &searcher.Item{
		Time:  &t,
		Level: searcher.LevelINFO,
		Line:  TiKVLogExample[1],
		Type:  searcher.TypeTiKV,
	}

	err = parser.Next()
	c.Assert(err, IsNil)
	item := parser.GetCurrentLog()
	c.Assert(item, DeepEquals, expect)

	err = parser.Next()
	c.Assert(err, Equals, io.EOF)
	item = parser.GetCurrentLog()
	c.Assert(item, IsNil)
}

var PDLogExample = []string{
	`2019/07/19 17:04:29.375 log.go:88: [info] mvcc: [store.index: compact 489789]`,
	`2019/07/19 17:04:29.375 log.go:86: [info] compactor: [Finished auto-compaction at revision 489789]`,
	`2019/07/19 17:04:29.376 log.go:88: [info] mvcc: [finished scheduled compaction at 489789 (took 957.288µs)]`,
}

func (a *AdaptorTestSuite) TestParsePDParser(c *C) {
	searchStr := "mvcc"
	logStr := strings.Join(PDLogExample, "\n")
	reader := bufio.NewReader(strings.NewReader(logStr))
	parser := searcher.PDsearcher{
		BaseParser: searcher.BaseParser{
			Reader:      reader,
			SearchBytes: []byte(searchStr),
		},
	}
	loc, err := time.LoadLocation("Asia/Chongqing")
	c.Assert(err, IsNil)

	t := time.Unix(1563527069, 375000000).In(loc)
	expect := &searcher.Item{
		Time:  &t,
		Level: searcher.LevelINFO,
		Line:  PDLogExample[0],
		Type:  searcher.TypePD,
	}

	err = parser.Next()
	c.Assert(err, IsNil)
	item := parser.GetCurrentLog()
	c.Assert(item, DeepEquals, expect)

	t = time.Unix(1563527069, 376000000).In(loc)
	expect = &searcher.Item{
		Time:  &t,
		Level: searcher.LevelINFO,
		Line:  PDLogExample[2],
		Type:  searcher.TypePD,
	}

	err = parser.Next()
	c.Assert(err, IsNil)
	item = parser.GetCurrentLog()
	c.Assert(item, DeepEquals, expect)

	err = parser.Next()
	c.Assert(err, Equals, io.EOF)
	item = parser.GetCurrentLog()
	c.Assert(item, IsNil)
}
