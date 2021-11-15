package sourcedata

import (
	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/errors"
	"strconv"
	"time"
)

type accumulator interface {
	feed([]string) error
}

type execTimeInfo struct {
	cnt                    int
	totalProcessTimeSecond float64
	lastOccurUnix          int64
}

func (info *execTimeInfo) update(cnt int, pTime float64, occurUnix int64) {
	info.cnt = +cnt
	info.totalProcessTimeSecond = +pTime
	if occurUnix > info.lastOccurUnix {
		info.lastOccurUnix = occurUnix
	}
}

func (info *execTimeInfo) getAvgProcessTime() float64 {
	if info.cnt == 0 {
		return 0
	}
	return info.totalProcessTimeSecond / float64(info.cnt)
}

type avgProcessTimePlanAccumulator struct {
	idxLookUp map[string]int
	data      map[string]map[string]*execTimeInfo
}

func (acc *avgProcessTimePlanAccumulator) updateExecTimeInfo(digest string, pDigest string, pTime float64, occurUnix int64) {
	if _, ok := acc.data[digest]; !ok {
		acc.data[digest] = map[string]*execTimeInfo{
			pDigest: {
				cnt:                    1,
				totalProcessTimeSecond: pTime,
				lastOccurUnix:          occurUnix,
			},
		}
		return
	}
	if _, ok := acc.data[digest][pDigest]; !ok {
		acc.data[digest][pDigest] = &execTimeInfo{
			cnt:                    1,
			totalProcessTimeSecond: pTime,
			lastOccurUnix:          occurUnix,
		}
		return
	}
	acc.data[digest][pDigest].update(1, pTime, occurUnix)
}

func (acc *avgProcessTimePlanAccumulator) feed(row []string) error {
	if len(acc.idxLookUp) < len(row) {
		return errors.New("invalid slow log row")
	}
	processKeys, err := strconv.Atoi(row[acc.idxLookUp["Process_keys"]])
	if err != nil && len(row[acc.idxLookUp["Process_keys"]]) > 0 {
		return err
	}
	if processKeys <= 1000000 {
		return nil
	}
	digest := row[acc.idxLookUp["Digest"]]
	planDigest := row[acc.idxLookUp["Plan_digest"]]
	processTime, err := strconv.ParseFloat(row[acc.idxLookUp["Process_time"]], 64)
	if err != nil && len(row[acc.idxLookUp["Process_time"]]) > 0 {
		return err
	}
	occurTime, err := time.Parse("2006-01-02 15:04:05", row[acc.idxLookUp["Time"]])
	if err != nil {
		return err
	}
	occurUnix := occurTime.Unix()
	acc.updateExecTimeInfo(digest, planDigest, processTime, occurUnix)
	return nil
}

func (acc *avgProcessTimePlanAccumulator) build() (map[string][2]proto.ExecutionPlanInfo, error) {
	result := make(map[string][2]proto.ExecutionPlanInfo)
	for digest, sql := range acc.data {
		if len(sql) == 1 {
			continue
		}
		var maxExePlanInfo proto.ExecutionPlanInfo
		var minExecPlanInfo proto.ExecutionPlanInfo
		for pDigest, plan := range sql {
			{
				avgProcessTime := int64(plan.getAvgProcessTime())
				if avgProcessTime > maxExePlanInfo.AvgProcessTime {
					maxExePlanInfo.AvgProcessTime = avgProcessTime
					maxExePlanInfo.PlanDigest = pDigest
					maxExePlanInfo.MaxLastTime = plan.lastOccurUnix
				}
			}
			{
				avgProcessTime := int64(plan.getAvgProcessTime())
				if avgProcessTime < minExecPlanInfo.AvgProcessTime {
					minExecPlanInfo.AvgProcessTime = avgProcessTime
					minExecPlanInfo.PlanDigest = pDigest
					minExecPlanInfo.MaxLastTime = plan.lastOccurUnix
				}
			}
		}
		result[digest] = [2]proto.ExecutionPlanInfo{
			minExecPlanInfo, maxExePlanInfo,
		}
	}
	return result, nil
}

type scanOldVersionPlanAccumulator struct {
	idxLookUp map[string]int
	data      map[string]map[string]struct{}
}

func (acc *scanOldVersionPlanAccumulator) feed(row []string) error {
	if len(acc.idxLookUp) < len(row) {
		return errors.New("invalid slow log row")
	}
	processKeys, err := strconv.Atoi(row[acc.idxLookUp["Process_keys"]])
	if err != nil && len(row[acc.idxLookUp["Process_keys"]]) > 0 {
		return err
	}
	if processKeys <= 10000 {
		return nil
	}
	totalKeys, err := strconv.Atoi(row[acc.idxLookUp["Total_keys"]])
	if err != nil && len(row[acc.idxLookUp["Total_keys"]]) > 0 {
		return err
	}
	if totalKeys < 2*processKeys {
		return nil
	}
	digest := row[acc.idxLookUp["Digest"]]
	planDigest := row[acc.idxLookUp["Plan_digest"]]
	if _, ok := acc.data[digest]; !ok {
		acc.data[digest] = map[string]struct{}{
			planDigest: {},
		}
	} else {
		acc.data[digest][planDigest] = struct{}{}
	}
	return nil
}

func (acc *scanOldVersionPlanAccumulator) build() ([]proto.DigestPair, error) {
	result := make([]proto.DigestPair, 0)
	for digest, sql := range acc.data {
		for pDigest := range sql {
			result = append(result, proto.DigestPair{
				Digest:     digest,
				PlanDigest: pDigest,
			})
		}
	}
	return result, nil
}

type skipDeletedCntPlanAccumulator struct {
	idxLookUp map[string]int
	data      map[string]map[string]struct{}
}

func (acc *skipDeletedCntPlanAccumulator) feed(row []string) error {
	if len(acc.idxLookUp) < len(row) {
		return errors.New("invalid slow log row")
	}
	processKeys, err := strconv.Atoi(row[acc.idxLookUp["Process_keys"]])
	if err != nil && len(row[acc.idxLookUp["Process_keys"]]) > 0 {
		return err
	}
	if processKeys <= 10000 {
		return nil
	}
	deleteSkippedCnt, err := strconv.Atoi(row[acc.idxLookUp["Rocksdb_delete_skipped_count"]])
	if err != nil && len(row[acc.idxLookUp["Rocksdb_delete_skipped_count"]]) > 0 {
		return err
	}
	delta := deleteSkippedCnt - processKeys
	if delta < 0 {
		delta = -delta
	}
	if delta > 1000 {
		return nil
	}
	digest := row[acc.idxLookUp["Digest"]]
	planDigest := row[acc.idxLookUp["Plan_digest"]]
	if _, ok := acc.data[digest]; !ok {
		acc.data[digest] = map[string]struct{}{
			planDigest: {},
		}
	} else {
		acc.data[digest][planDigest] = struct{}{}
	}
	return nil
}

func (acc *skipDeletedCntPlanAccumulator) build() ([]proto.DigestPair, error) {
	result := make([]proto.DigestPair, 0)
	for digest, sql := range acc.data {
		for pDigest := range sql {
			result = append(result, proto.DigestPair{
				Digest:     digest,
				PlanDigest: pDigest,
			})
		}
	}
	return result, nil
}

func NewIdxLookup(header []string) map[string]int {
	lookup := make(map[string]int)
	for idx, val := range header {
		lookup[val] = idx
	}
	return lookup
}
