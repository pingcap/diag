package task

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	SlowLogTimeFormat    = time.RFC3339Nano
	OldSlowLogTimeFormat = "2006-01-02-15:04:05.999999999 -0700"
)

const (
	// SlowLogRowPrefixStr is slow log row prefix.
	SlowLogRowPrefixStr = "# "
	// SlowLogSpaceMarkStr is slow log space mark.
	SlowLogSpaceMarkStr = ": "
	// SlowLogSQLSuffixStr is slow log suffix.
	SlowLogSQLSuffixStr = ";"
	// SlowLogTimeStr is slow log field name.
	SlowLogTimeStr = "Time"
	// SlowLogStartPrefixStr is slow log start row prefix.
	SlowLogStartPrefixStr = SlowLogRowPrefixStr + SlowLogTimeStr + SlowLogSpaceMarkStr
	// SlowLogTxnStartTSStr is slow log field name.
	SlowLogTxnStartTSStr = "Txn_start_ts"
	// SlowLogUserStr is slow log field name.
	SlowLogUserStr = "User"
	// SlowLogHostStr only for slow_query table usage.
	SlowLogHostStr = "Host"
	// SlowLogConnIDStr is slow log field name.
	SlowLogConnIDStr = "Conn_ID"
	// SlowLogQueryTimeStr is slow log field name.
	SlowLogQueryTimeStr = "Query_time"
	// SlowLogDBStr is slow log field name.
	SlowLogDBStr = "DB"
	// SlowLogIsInternalStr is slow log field name.
	SlowLogIsInternalStr = "Is_internal"
	// SlowLogIndexIDsStr is slow log field name.
	SlowLogIndexIDsStr = "Index_ids"
	// SlowLogDigestStr is slow log field name.
	SlowLogDigestStr = "Digest"
	// SlowLogQuerySQLStr is slow log field name.
	SlowLogQuerySQLStr = "Query" // use for slow log table, slow log will not print this field name but print sql directly.
	// SlowLogStatsInfoStr is plan stats info.
	SlowLogStatsInfoStr = "Stats"
	// SlowLogNumCopTasksStr is the number of cop-tasks.
	SlowLogNumCopTasksStr = "Num_cop_tasks"
	// SlowLogCopProcAvg is the average process time of all cop-tasks.
	SlowLogCopProcAvg = "Cop_proc_avg"
	// SlowLogCopProcP90 is the p90 process time of all cop-tasks.
	SlowLogCopProcP90 = "Cop_proc_p90"
	// SlowLogCopProcMax is the max process time of all cop-tasks.
	SlowLogCopProcMax = "Cop_proc_max"
	// SlowLogCopProcAddr is the address of TiKV where the cop-task which cost max process time run.
	SlowLogCopProcAddr = "Cop_proc_addr"
	// SlowLogCopWaitAvg is the average wait time of all cop-tasks.
	SlowLogCopWaitAvg = "Cop_wait_avg"
	// SlowLogCopWaitP90 is the p90 wait time of all cop-tasks.
	SlowLogCopWaitP90 = "Cop_wait_p90"
	// SlowLogCopWaitMax is the max wait time of all cop-tasks.
	SlowLogCopWaitMax = "Cop_wait_max"
	// SlowLogCopWaitAddr is the address of TiKV where the cop-task which cost wait process time run.
	SlowLogCopWaitAddr = "Cop_wait_addr"
	// SlowLogMemMax is the max number bytes of memory used in this statement.
	SlowLogMemMax = "Mem_max"
	// SlowLogSucc is used to indicate whether this sql execute successfully.
	SlowLogSucc = "Succ"
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

type SaveSlowLogTask struct {
	BaseTask
}

func SaveSlowLogInfo(base BaseTask) Task {
	return &SaveSlowLogTask{base}
}

func (t *SaveSlowLogTask) Run() error {
	if !t.data.args.Collect(ITEM_LOG) || t.data.status[ITEM_LOG].Status != "success" {
		return nil
	}

	logDir := filepath.Join(t.src, "log")

	files, err := LoadSlowQueryLogFiles(logDir)
	if err != nil {
		return err
	}
	tz, err := time.LoadLocation("Asia/Chongqing")
	if err != nil {
		return err
	}
	for _, file := range files {
		err := t.InsertSlowLog(file, tz)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *SaveSlowLogTask) InsertSlowLog(file SlowQueryLogFile, tz *time.Location) error {
	f, err := os.Open(file.path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	querys, err := ParseSlowLog(tz, r)
	if err != nil {
		return err
	}
	for _, q := range querys {
		if _, err := t.db.Exec(
			`INSERT INTO inspection_slow_log(inspection, instance, time, txn_start_ts, user, conn_id, query_time, db, digest, query, node_ip) 
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, t.inspectionId, file.instance, q.time, q.txnStartTs, q.user, q.connID, q.queryTime, q.db, q.digest, q.sql, file.host,
		); err != nil {
			fmt.Printf("t.inspectionId=%s,file.instance=%s,q.digest=%s\n", t.inspectionId, file.instance, q.digest)
			log.Error("db.Exec: ", err)
			return err
		}
	}
	return nil
}

type SlowQueryLogFile struct {
	instance string
	host     string
	path     string
}

func LoadSlowQueryLogFiles(logDir string) ([]SlowQueryLogFile, error) {
	var files []SlowQueryLogFile
	// "xxxxx", "172.16.5.7", "tidb-4000", "tidb_slow_query.log")
	err := filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error("walk dir:", err)
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

func ParseSlowLog(tz *time.Location, reader *bufio.Reader) ([]slowQueryTuple, error) {
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
		if !startFlag && strings.HasPrefix(line, SlowLogStartPrefixStr) {
			st = &slowQueryTuple{}
			err = st.setFieldValue(tz, SlowLogTimeStr, line[len(SlowLogStartPrefixStr):])
			if err != nil {
				return rows, err
			}
			startFlag = true
			continue
		}

		if startFlag {
			// Parse slow log field.
			if strings.HasPrefix(line, SlowLogRowPrefixStr) {
				line = line[len(SlowLogRowPrefixStr):]
				fieldValues := strings.Split(line, " ")
				for i := 0; i < len(fieldValues)-1; i += 2 {
					field := fieldValues[i]
					if strings.HasSuffix(field, ":") {
						field = field[:len(field)-1]
					}
					err = st.setFieldValue(tz, field, fieldValues[i+1])
					if err != nil {
						return rows, err
					}
				}
			} else if strings.HasSuffix(line, SlowLogSQLSuffixStr) {
				// Get the sql string, and mark the start flag to false.
				err = st.setFieldValue(tz, SlowLogQuerySQLStr, string(line))
				if err != nil {
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
	case SlowLogTimeStr:
		t, err := ParseTime(value)
		if err != nil {
			return err
		}
		if t.Location() != tz {
			t = t.In(tz)
		}
		st.time = t
	case SlowLogTxnStartTSStr:
		num, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		st.txnStartTs = num
	case SlowLogUserStr:
		fields := strings.SplitN(value, "@", 2)
		if len(field) > 0 {
			st.user = fields[0]
		}
		if len(field) > 1 {
			st.host = fields[1]
		}
	case SlowLogConnIDStr:
		num, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		st.connID = num
	case SlowLogQueryTimeStr:
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
	case SlowLogDBStr:
		st.db = value
	case SlowLogIndexIDsStr:
		st.indexIDs = value
	case SlowLogIsInternalStr:
		st.isInternal = value == "true"
	case SlowLogDigestStr:
		st.digest = value
	case SlowLogStatsInfoStr:
		st.statsInfo = value
	case SlowLogCopProcAvg:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.avgProcessTime = num
	case SlowLogCopProcP90:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.p90ProcessTime = num
	case SlowLogCopProcMax:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.maxProcessTime = num
	case SlowLogCopProcAddr:
		st.maxProcessAddress = value
	case SlowLogCopWaitAvg:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.avgWaitTime = num
	case SlowLogCopWaitP90:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.p90WaitTime = num
	case SlowLogCopWaitMax:
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		st.maxWaitTime = num
	case SlowLogCopWaitAddr:
		st.maxWaitAddress = value
	case SlowLogMemMax:
		num, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		st.memMax = num
	case SlowLogSucc:
		succ, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		st.succ = succ
	case SlowLogQuerySQLStr:
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
	t, err := time.Parse(SlowLogTimeFormat, s)
	if err != nil {
		// This is for compatibility.
		t, err = time.Parse(OldSlowLogTimeFormat, s)
		if err != nil {
			err = fmt.Errorf("string \"%v\" doesn't has a prefix that matches format \"%v\", err: %v", s, SlowLogTimeFormat, err)
		}
	}
	return t, err
}
