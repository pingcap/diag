package sourcedata

import "testing"

func TestAvgProcessTimePlanAccumulator(t *testing.T) {
	rows := [][]string{
		{"2021-10-29 16:47:46.491841", "1100000", "1", "a", "a1"},
		{"2021-10-29 16:47:47.491841", "1100000", "2", "a", "a2"},
		{"2021-10-29 16:47:48.491841", "1100000", "3", "a", "a3"},
		{"2021-10-29 16:47:49.491841", "1100000", "10", "b", "b1"},
		{"2021-10-29 16:47:50.491841", "1100000", "20", "b", "b2"},
		{"2021-10-29 16:47:51.491841", "1100000", "100", "b", "b2"},
		{"2021-10-29 16:47:51.5", "1100000", "100", "c", "c1"},
	}
	idxLookUp := map[string]int{
		"Time":         0,
		"Process_keys": 1,
		"Process_time": 2,
		"Digest":       3,
		"Plan_digest":  4,
	}
	acc, err := NewAvgProcessTimePlanAccumulator(idxLookUp)
	if err != nil {
		t.Error(err)
	}
	for _, row := range rows {
		if err := acc.feed(row); err != nil {
			t.Error(err)
		}
	}
	info, err := acc.build()
	if err != nil {
		t.Error(err)
	}
	if len(info) != 2 {
		t.Error("wrong result")
	}
	if sqlInfo, ok := info["a"]; !ok {
		t.Error("wrong result")
	} else {
		if sqlInfo[0].PlanDigest != "a1" && sqlInfo[0].AvgProcessTime != int64(1) {
			t.Error("wrong result")
		}
		if sqlInfo[1].PlanDigest != "a3" && sqlInfo[1].AvgProcessTime != int64(3) {
			t.Error("wrong result")
		}
	}
	if sqlInfo, ok := info["b"]; !ok {
		t.Error("wrong result")
	} else {
		if sqlInfo[0].PlanDigest != "b1" && sqlInfo[0].AvgProcessTime != int64(10) {
			t.Error("wrong result")
		}
		if sqlInfo[1].PlanDigest != "b2" && sqlInfo[1].AvgProcessTime != int64(60) {
			t.Error("wrong result")
		}
	}
	if _, ok := info["c"]; ok {
		t.Error("wrong result")
	}
}

func TestScanOldVersionPlanAccumulator(t *testing.T) {
	idxLookUp := map[string]int{
		"Digest":       0,
		"Plan_digest":  1,
		"Process_keys": 2,
		"Total_keys":   3,
	}
	rows := [][]string{
		{"a", "a1", "11000", "110000"},
		{"a", "a2", "11000", "110000"},
		{"a", "a3", "11000", "110000"},
		{"b", "b1", "11000", "110000"},
		{"b", "b2", "11000", "110000"},
		{"b", "b2", "10", "10"},
		{"c", "c1", "11000", "15000"},
	}
	acc, err := NewScanOldVersionPlanAccumulator(idxLookUp)
	if err != nil {
		t.Error(err)
	}
	for _, row := range rows {
		if err := acc.feed(row); err != nil {
			t.Error()
		}
	}
	info, err := acc.build()
	if err != nil {
		t.Error(err)
	}
	if len(info) != 5 {
		t.Error("wrong result")
	}
}

func TestSkipDeletedCntPlanAccumulator(t *testing.T) {
	idxLookUp := map[string]int{
		"Digest":                       0,
		"Plan_digest":                  1,
		"Process_keys":                 2,
		"Rocksdb_delete_skipped_count": 3,
	}
	rows := [][]string{
		{"a", "a1", "100", "110000"},
		{"a", "a2", "21000", "21001"},
		{"a", "a3", "11000", "15000"},
		{"b", "b1", "11000", "11100"},
		{"b", "b2", "11000", "110000"},
		{"b", "b2", "10", "10"},
		{"c", "c1", "11000", "12000"},
	}
	acc, err := NewSkipDeletedCntPlanAccumulator(idxLookUp)
	if err != nil {
		t.Error(err)
	}
	for _, row := range rows {
		if err := acc.feed(row); err != nil {
			t.Error()
		}
	}
	info, err := acc.build()
	if err != nil {
		t.Error(err)
	}
	if len(info) != 3 {
		t.Error("wrong result")
	}
}
