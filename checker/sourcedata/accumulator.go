package sourcedata

import (
	"encoding/csv"
	"fmt"
	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/errors"
	"github.com/pingcap/log"
	"go.uber.org/zap"
	"io"
	"math"
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

func (info *execTimeInfo) state() string {
	return fmt.Sprintf("cnt %+v, totalProcessTimeSecond %+v, lastOccurUnix %+v", info.cnt, info.totalProcessTimeSecond, info.lastOccurUnix)
}

func (info *execTimeInfo) update(cnt int, pTime float64, occurUnix int64) {
	info.cnt =+ cnt
	info.totalProcessTimeSecond =+ pTime
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
	csvWriter *csv.Writer
	filterOutCnt int64
}

func (acc *avgProcessTimePlanAccumulator) setCSVWriter(w io.Writer) {
	acc.csvWriter = csv.NewWriter(w)
}

// header must contain required fileds
func NewAvgProcessTimePlanAccumulator(idxLookUp map[string]int) (*avgProcessTimePlanAccumulator, error) {
	if _, ok := idxLookUp["Process_keys"]; !ok {
		return nil, errors.New("idxLookUp must contain Process_keys")
	}
	if _, ok := idxLookUp["Process_time"]; !ok {
		return nil, errors.New("idxLookUp must contain Process_time")
	}
	if _, ok := idxLookUp["Time"]; !ok {
		return nil, errors.New("idxLookUp must contain Time")
	}

	if _, ok := idxLookUp["Digest"]; !ok {
		return nil, errors.New("idxLookUp must contain Digest")
	}

	if _, ok := idxLookUp["Plan_digest"]; !ok {
		return nil, errors.New("idxLookUp must contain Plan_digest")
	}
	return &avgProcessTimePlanAccumulator{
		idxLookUp: idxLookUp,
		data:      make(map[string]map[string]*execTimeInfo),
	}, nil
}

func (acc *avgProcessTimePlanAccumulator) updateExecTimeInfo(digest string, pDigest string, pTime float64, occurUnix int64) {
	if _, ok := acc.data[digest]; !ok {
		acc.data[digest] = map[string]*execTimeInfo{
			pDigest: &execTimeInfo{
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
	if len(acc.idxLookUp) > len(row) {
		return errors.New("invalid slow log row")
	}
	processKeys, err := strconv.Atoi(row[acc.idxLookUp["Process_keys"]])
	if err != nil && len(row[acc.idxLookUp["Process_keys"]]) > 0 {
		return err
	}
	if processKeys <= 1000000 {
		acc.filterOutCnt++
		return nil
	}
	digest := row[acc.idxLookUp["Digest"]]
	planDigest := row[acc.idxLookUp["Plan_digest"]]
	processTime, err := strconv.ParseFloat(row[acc.idxLookUp["Process_time"]], 64)
	if err != nil && len(row[acc.idxLookUp["Process_time"]]) > 0 {
		return err
	}
	occurTime, err := time.Parse("2006-01-02 15:04:05.999999", row[acc.idxLookUp["Time"]])
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
		minExecPlanInfo.AvgProcessTime = math.MaxInt64
		for pDigest, plan := range sql {
			avgProcessTime := int64(plan.getAvgProcessTime())
			{
				if avgProcessTime > maxExePlanInfo.AvgProcessTime {
					maxExePlanInfo.AvgProcessTime = avgProcessTime
					maxExePlanInfo.PlanDigest = pDigest
					maxExePlanInfo.MaxLastTime = plan.lastOccurUnix
				}
			}
			{
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
	if acc.csvWriter != nil {
		defer acc.csvWriter.Flush()
		if err := acc.csvWriter.Write([]string{"Digest", "Plan_digest","avg_process_time", "last_time"}); err != nil {
			return nil, err
		}
		for digest, planDigest := range result {
			{
				row := []string{digest, planDigest[0].PlanDigest, strconv.FormatInt(planDigest[0].AvgProcessTime, 10), strconv.FormatInt(planDigest[0].MaxLastTime, 10)}
				if err := acc.csvWriter.Write(row); err != nil {
					return nil, err
				}
			}
			{
				row := []string{digest, planDigest[1].PlanDigest, strconv.FormatInt(planDigest[1].AvgProcessTime, 10), strconv.FormatInt(planDigest[1].MaxLastTime, 10)}
				if err := acc.csvWriter.Write(row); err != nil {
					return nil, err
				}
			}
		}
	}
	return result, nil
}

func (acc *avgProcessTimePlanAccumulator) debugState() {
	log.Debug("avgProcessTimePlanAccumulator", zap.Any("filterOutCnt", acc.filterOutCnt))
	for sqlDigest, planDigest := range acc.data {
		for pDigest, info := range planDigest {
			log.Debug("avgProcessTimePlanAccumulator", zap.String("digest", sqlDigest),
				zap.String("planDigest", pDigest),
				zap.String("execInfo", info.state()))
		}
	}
}

type scanOldVersionPlanAccumulator struct {
	idxLookUp map[string]int
	data      map[string]map[string]struct{}
	csvWriter *csv.Writer
	filterOutCnt int64
}

// header must contain Process_keys, Process_time, Time
func NewScanOldVersionPlanAccumulator(idxLookUp map[string]int) (*scanOldVersionPlanAccumulator, error) {
	if _, ok := idxLookUp["Process_keys"]; !ok {
		return nil, errors.New("idxLookUp must contain Process_keys")
	}
	if _, ok := idxLookUp["Total_keys"]; !ok {
		return nil, errors.New("idxLookUp must contain Total_keys")
	}
	if _, ok := idxLookUp["Digest"]; !ok {
		return nil, errors.New("idxLookUp must contain Digest")
	}
	if _, ok := idxLookUp["Plan_digest"]; !ok {
		return nil, errors.New("idxLookUp must contain Plan_digest")
	}
	return &scanOldVersionPlanAccumulator{
		idxLookUp: idxLookUp,
		data:      make(map[string]map[string]struct{}),
	}, nil
}

func (acc *scanOldVersionPlanAccumulator) setCSVWriter(w io.Writer) {
	acc.csvWriter = csv.NewWriter(w)
}

func (acc *scanOldVersionPlanAccumulator) feed(row []string) error {
	if len(acc.idxLookUp) > len(row) {
		return errors.New("invalid slow log row")
	}
	processKeys, err := strconv.Atoi(row[acc.idxLookUp["Process_keys"]])
	if err != nil && len(row[acc.idxLookUp["Process_keys"]]) > 0 {
		return err
	}
	if processKeys <= 10000 {
		acc.filterOutCnt++
		return nil
	}
	totalKeys, err := strconv.Atoi(row[acc.idxLookUp["Total_keys"]])
	if err != nil && len(row[acc.idxLookUp["Total_keys"]]) > 0 {
		return err
	}
	if totalKeys < 2*processKeys {
		acc.filterOutCnt++
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
	if acc.csvWriter != nil {
		defer acc.csvWriter.Flush()
		if err := acc.csvWriter.Write([]string{"Digest", "Plan_digest"}); err != nil {
			return nil, err
		}
		for _, sqlInfo := range result {
			{
				row := []string{sqlInfo.Digest, sqlInfo.PlanDigest}
				if err := acc.csvWriter.Write(row); err != nil {
					return nil, err
				}
			}
		}
	}
	return result, nil
}

func (acc *scanOldVersionPlanAccumulator) debugState() {
	log.Debug("scanOldVersionPlanAccumulator", zap.Any("filterOutCnt", acc.filterOutCnt))
	for sqlDigest, plans := range acc.data {
		for pDigest, _ := range plans {
			log.Debug("scanOldVersionPlanAccumulator", zap.String("digest", sqlDigest),
				zap.String("planDigest", pDigest))
		}
	}
}

type skipDeletedCntPlanAccumulator struct {
	idxLookUp map[string]int
	data      map[string]map[string]struct{}
	csvWriter *csv.Writer
	filterOutCnt int64
}

func NewSkipDeletedCntPlanAccumulator(header map[string]int) (*skipDeletedCntPlanAccumulator, error) {
	if _, ok := header["Process_keys"]; !ok {
		return nil, errors.New("header must contain Process_keys")
	}
	if _, ok := header["Rocksdb_delete_skipped_count"]; !ok {
		return nil, errors.New("header must contain Rocksdb_delete_skipped_count")
	}
	if _, ok := header["Digest"]; !ok {
		return nil, errors.New("header must contain Digest")
	}
	if _, ok := header["Plan_digest"]; !ok {
		return nil, errors.New("header must contain Plan_digest")
	}
	return &skipDeletedCntPlanAccumulator{
		idxLookUp: header,
		data:      make(map[string]map[string]struct{}),
	}, nil
}

func (acc *skipDeletedCntPlanAccumulator) setCSVWriter(w io.Writer) {
	acc.csvWriter = csv.NewWriter(w)
}

func (acc *skipDeletedCntPlanAccumulator) feed(row []string) error {
	if len(acc.idxLookUp) > len(row) {
		return errors.New("invalid slow log row")
	}
	processKeys, err := strconv.Atoi(row[acc.idxLookUp["Process_keys"]])
	if err != nil && len(row[acc.idxLookUp["Process_keys"]]) > 0 {
		return err
	}
	if processKeys <= 10000 {
		acc.filterOutCnt++
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
		acc.filterOutCnt++
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
	if acc.csvWriter != nil {
		defer acc.csvWriter.Flush()
		if err := acc.csvWriter.Write([]string{"Digest", "Plan_digest"}); err != nil {
			return nil, err
		}
		for _, sqlInfo := range result {
			{
				row := []string{sqlInfo.Digest, sqlInfo.PlanDigest}
				if err := acc.csvWriter.Write(row); err != nil {
					return nil, err
				}
			}
		}
	}
	return result, nil
}

func (acc *skipDeletedCntPlanAccumulator) debugState() {
	log.Debug("skipDeletedCntPlanAccumulator", zap.Any("filterOutCnt", acc.filterOutCnt))
	for sqlDigest, plans := range acc.data {
		for pDigest, _ := range plans {
			log.Debug("skipDeletedCntPlanAccumulator", zap.String("digest", sqlDigest),
				zap.String("planDigest", pDigest))
		}
	}
}

func NewIdxLookup(header []string) map[string]int {
	lookup := make(map[string]int)
	for idx, val := range header {
		lookup[val] = idx
	}
	return lookup
}
