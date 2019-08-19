package logs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type slowQueryTuple struct {
	time              time.Time
	txnStartTs        uint64
	user              string
	host              string
	connID            uint64
	queryTime         float64
	processTime       float64
	waitTime          float64
	backOffTime       float64
	requestCount      uint64
	totalKeys         uint64
	processKeys       uint64
	db                string
	indexIDs          string
	isInternal        bool
	digest            string
	statsInfo         string
	avgProcessTime    float64
	p90ProcessTime    float64
	maxProcessTime    float64
	maxProcessAddress string
	avgWaitTime       float64
	p90WaitTime       float64
	maxWaitTime       float64
	maxWaitAddress    string
	memMax            int64
	succ              bool
	sql               string
}

const (
	SlowQueryTimeFormat    = time.RFC3339Nano
	OldSlowQueryTimeFormat = "2006-01-02-15:04:05.999999999 -0700"
)

const (
	// SlowQueryRowPrefixStr is slow log row prefix.
	SlowQueryRowPrefixStr = "# "
	// SlowQuerySpaceMarkStr is slow log space mark.
	SlowQuerySpaceMarkStr = ": "
	// SlowQuerySQLSuffixStr is slow log suffix.
	SlowQuerySQLSuffixStr = ";"
	// SlowQueryTimeStr is slow log field name.
	SlowQueryTimeStr = "Time"
	// SlowQueryStartPrefixStr is slow log start row prefix.
	SlowQueryStartPrefixStr = SlowQueryRowPrefixStr + SlowQueryTimeStr + SlowQuerySpaceMarkStr
	// SlowQueryTxnStartTSStr is slow log field name.
	SlowQueryTxnStartTSStr = "Txn_start_ts"
	// SlowQueryUserStr is slow log field name.
	SlowQueryUserStr = "User"
	// SlowQueryHostStr only for slow_query table usage.
	SlowQueryHostStr = "Host"
	// SlowQueryConnIDStr is slow log field name.
	SlowQueryConnIDStr = "Conn_ID"
	// SlowQueryQueryTimeStr is slow log field name.
	SlowQueryQueryTimeStr = "Query_time"
	// SlowQueryDBStr is slow log field name.
	SlowQueryDBStr = "DB"
	// SlowQueryIsInternalStr is slow log field name.
	SlowQueryIsInternalStr = "Is_internal"
	// SlowQueryIndexIDsStr is slow log field name.
	SlowQueryIndexIDsStr = "Index_ids"
	// SlowQueryDigestStr is slow log field name.
	SlowQueryDigestStr = "Digest"
	// SlowQueryQuerySQLStr is slow log field name.
	SlowQueryQuerySQLStr = "Query" // use for slow log table, slow log will not print this field name but print sql directly.
	// SlowQueryStatsInfoStr is plan stats info.
	SlowQueryStatsInfoStr = "Stats"
	// SlowQueryNumCopTasksStr is the number of cop-tasks.
	SlowQueryNumCopTasksStr = "Num_cop_tasks"
	// SlowQueryCopProcAvg is the average process time of all cop-tasks.
	SlowQueryCopProcAvg = "Cop_proc_avg"
	// SlowQueryCopProcP90 is the p90 process time of all cop-tasks.
	SlowQueryCopProcP90 = "Cop_proc_p90"
	// SlowQueryCopProcMax is the max process time of all cop-tasks.
	SlowQueryCopProcMax = "Cop_proc_max"
	// SlowQueryCopProcAddr is the address of TiKV where the cop-task which cost max process time run.
	SlowQueryCopProcAddr = "Cop_proc_addr"
	// SlowQueryCopWaitAvg is the average wait time of all cop-tasks.
	SlowQueryCopWaitAvg = "Cop_wait_avg"
	// SlowQueryCopWaitP90 is the p90 wait time of all cop-tasks.
	SlowQueryCopWaitP90 = "Cop_wait_p90"
	// SlowQueryCopWaitMax is the max wait time of all cop-tasks.
	SlowQueryCopWaitMax = "Cop_wait_max"
	// SlowQueryCopWaitAddr is the address of TiKV where the cop-task which cost wait process time run.
	SlowQueryCopWaitAddr = "Cop_wait_addr"
	// SlowQueryMemMax is the max number bytes of memory used in this statement.
	SlowQueryMemMax = "Mem_max"
	// SlowQuerySucc is used to indicate whether this sql execute successfully.
	SlowQuerySucc = "Succ"
)

const (
	// ProcessTimeStr represents the sum of process time of all the coprocessor tasks.
	ProcessTimeStr = "Process_time"
	// WaitTimeStr means the time of all coprocessor wait.
	WaitTimeStr = "Wait_time"
	// BackoffTimeStr means the time of all back-off.
	BackoffTimeStr = "Backoff_time"
	// RequestCountStr means the request count.
	RequestCountStr = "Request_count"
	// TotalKeysStr means the total scan keys.
	TotalKeysStr = "Total_keys"
	// ProcessKeysStr means the total processed keys.
	ProcessKeysStr = "Process_keys"
)

type saveSlowQueryTask struct{}

func SaveSlowQuery() *saveSlowQueryTask {
	return &saveSlowQueryTask{}
}

// Save slow query into database.
func (t *saveSlowQueryTask) Run(c *boot.Config, m *boot.Model) {
	logDir := filepath.Join(c.Src, "log")

	files, err := loadSlowQueryLogFiles(logDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error(err)
		}
		return
	}
	tz, err := time.LoadLocation("Asia/Chongqing")
	if err != nil {
		return
	}
	for _, file := range files {
		t.InsertSlowQuery(m, c.InspectionId, file, tz)
	}
}

func (t *saveSlowQueryTask) InsertSlowQuery(m *boot.Model, inspectionId string, file SlowQueryLogFile, tz *time.Location) {
	f, err := os.Open(file.path)
	if err != nil {
		log.Error("open log file:", err)
		return
	}
	defer f.Close()
	r := bufio.NewReader(f)
	querys, err := ParseSlowQuery(tz, r)
	if err != nil {
		log.Error("parse slow log:", err)
		return
	}
	for _, q := range querys {
		if err := m.InsertInspectionSlowLog(&model.SlowLogInfo{
			InspectionId: inspectionId,
			Time:         q.time,
			Query:        q.sql,
		}); err != nil {
			log.Error("insert slow log info:", err)
		}
	}
}

type SlowQueryLogFile struct {
	instance string
	host     string
	path     string
}

func loadSlowQueryLogFiles(logDir string) ([]SlowQueryLogFile, error) {
	var files []SlowQueryLogFile
	// "xxxxx", "172.16.5.7", "tidb-4000", "tidb_slow_query.log")
	err := filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() != "tidb_slow_query.log" {
			return nil
		}
		paths := strings.Split(path, string(filepath.Separator))
		if len(paths) < 4 {
			return fmt.Errorf("wrong slow query log file path: %v\n", paths)
		}
		file := SlowQueryLogFile{
			instance: paths[len(paths)-4],
			host:     paths[len(paths)-3],
			path:     path,
		}
		files = append(files, file)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func ParseSlowQuery(tz *time.Location, reader *bufio.Reader) ([]slowQueryTuple, error) {
	var rows []slowQueryTuple
	startFlag := false
	var st *slowQueryTuple
	for {
		lineByte, err := getOneLine(reader)
		if err != nil {
			if err == io.EOF {
				return rows, nil
			}
			return rows, err
		}
		line := string(lineByte)
		// Check slow log entry start flag.
		if !startFlag && strings.HasPrefix(line, SlowQueryStartPrefixStr) {
			st = &slowQueryTuple{}
			if err := st.setFieldValue(tz, SlowQueryTimeStr, line[len(SlowQueryStartPrefixStr):]); err != nil {
				return rows, err
			}
			startFlag = true
			continue
		}

		if startFlag {
			// Parse slow log field.
			if strings.HasPrefix(line, SlowQueryRowPrefixStr) {
				line = line[len(SlowQueryRowPrefixStr):]
				fieldValues := strings.Split(line, " ")
				for i := 0; i < len(fieldValues)-1; i += 2 {
					field := fieldValues[i]
					if strings.HasSuffix(field, ":") {
						field = field[:len(field)-1]
					}
					if err := st.setFieldValue(tz, field, fieldValues[i+1]); err != nil {
						return rows, err
					}
				}
			} else if strings.HasSuffix(line, SlowQuerySQLSuffixStr) {
				// Get the sql string, and mark the start flag to false.
				if err := st.setFieldValue(tz, SlowQueryQuerySQLStr, string(line)); err != nil {
					return rows, err
				}
				rows = append(rows, *st)
				startFlag = false
			} else {
				startFlag = false
			}
		}
	}
}

func (st *slowQueryTuple) setFieldValue(tz *time.Location, field, value string) error {
	switch field {
	case SlowQueryTimeStr:
		t, err := ParseTime(value)
		if err != nil {
			return err
		}
		if t.Location() != tz {
			t = t.In(tz)
		}
		st.time = t
	case SlowQueryTxnStartTSStr:
		num, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		st.txnStartTs = num
	case SlowQueryUserStr:
		fields := strings.SplitN(value, "@", 2)
		if len(field) > 0 {
			st.user = fields[0]
		}
		if len(field) > 1 {
			st.host = fields[1]
		}
	case SlowQueryConnIDStr:
		num, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		st.connID = num
	case SlowQueryQueryTimeStr:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.queryTime = num
	case ProcessTimeStr:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.processTime = num
	case WaitTimeStr:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.waitTime = num
	case BackoffTimeStr:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.backOffTime = num
	case RequestCountStr:
		num, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		st.requestCount = num
	case TotalKeysStr:
		num, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		st.totalKeys = num
	case ProcessKeysStr:
		num, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		st.processKeys = num
	case SlowQueryDBStr:
		st.db = value
	case SlowQueryIndexIDsStr:
		st.indexIDs = value
	case SlowQueryIsInternalStr:
		st.isInternal = value == "true"
	case SlowQueryDigestStr:
		st.digest = value
	case SlowQueryStatsInfoStr:
		st.statsInfo = value
	case SlowQueryCopProcAvg:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.avgProcessTime = num
	case SlowQueryCopProcP90:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.p90ProcessTime = num
	case SlowQueryCopProcMax:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.maxProcessTime = num
	case SlowQueryCopProcAddr:
		st.maxProcessAddress = value
	case SlowQueryCopWaitAvg:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.avgWaitTime = num
	case SlowQueryCopWaitP90:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.p90WaitTime = num
	case SlowQueryCopWaitMax:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.maxWaitTime = num
	case SlowQueryCopWaitAddr:
		st.maxWaitAddress = value
	case SlowQueryMemMax:
		num, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		st.memMax = num
	case SlowQuerySucc:
		succ, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		st.succ = succ
	case SlowQueryQuerySQLStr:
		st.sql = value
	}
	return nil
}

const MaxOfMaxAllowedPacket = 65536

func getOneLine(reader *bufio.Reader) ([]byte, error) {
	lineByte, isPrefix, err := reader.ReadLine()
	if err != nil {
		return lineByte, err
	}
	var tempLine []byte
	for isPrefix {
		tempLine, isPrefix, err = reader.ReadLine()
		lineByte = append(lineByte, tempLine...)

		// Use the max value of max_allowed_packet to check the single line length.
		if len(lineByte) > int(MaxOfMaxAllowedPacket) {
			return lineByte, fmt.Errorf("single line length exceeds limit: %v", MaxOfMaxAllowedPacket)
		}
		if err != nil {
			return lineByte, err
		}
	}
	return lineByte, err
}

func ParseTime(s string) (time.Time, error) {
	t, err := time.Parse(SlowQueryTimeFormat, s)
	if err != nil {
		// This is for compatibility.
		t, err = time.Parse(OldSlowQueryTimeFormat, s)
		if err != nil {
			err = fmt.Errorf("string \"%v\" doesn't has a prefix that matches format \"%v\", err: %v", s, SlowQueryTimeFormat, err)
		}
	}
	return t, err
}
