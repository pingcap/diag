package sourcedata

import (
	"context"
	"testing"
	"time"
)

func TestSlowQueryRetriever(t *testing.T) {
	loc := time.Local
	retriever := slowQueryRetriever{
		concurrency:           1,
		timeZone:              loc,
		outputCols:            []string{"Digest", "Plan_digest", "Process_time"},
		desc:                  false,
		filterTimeRanges:      false,
		singleFile:            true,
		slowQueryFile:         "../testdata/tidb_slow_query.log",
	}
	defer retriever.Close()
	result := make([][]string,0)
	var end bool
	for {
		if end {
			break
		}
		data, err := retriever.retrieve(context.Background())
		if err != nil {
			t.Error(err)
		}
		if len(data) == 0{
			end = true
		}else {
			for _, v := range data {
				result = append(result, v)
			}
		}
	}
	if len(result) == 0 {
		t.Error("result should not empty")
	}
	for _, row := range result{
		t.Logf("%#v\n",row)
	}
}
