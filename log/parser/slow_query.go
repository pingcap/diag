package parser

import (
	"regexp"
	"time"

	"github.com/pingcap/diag/log/item"
)

// Parse tidb slow query log, example:
// # Time: 2019-08-22T10:46:31.81833097+08:00
// # Txn_start_ts: 410633369355550820
// # User: root@127.0.0.1
// # Conn_ID: 2073
// # Query_time: 0.522346068
// # DB: sbtest
// # Is_internal: false
// # Digest: 24e32e2ec145a5ce9632abaa8ebfa009481a9f779c9ad50f4cf353fc8a7f63f7
// # Num_cop_tasks: 0
// # Cop_proc_avg: 0 Cop_proc_p90: 0 Cop_proc_max: 0
type SlowQueryParser struct{}

var (
	SlowQueryRE = regexp.MustCompile("^# Time: (.*)$")
)

func (*SlowQueryParser) ParseHead(head []byte) (*time.Time, item.LevelType) {
	if !SlowQueryRE.Match(head) {
		return nil, item.LevelInvalid
	}
	m := SlowQueryRE.FindSubmatch(head)
	t, err := time.Parse(time.RFC3339Nano, string(m[1]))
	if err != nil {
		return nil, item.LevelInvalid
	}
	return &t, item.LevelInvalid
}
