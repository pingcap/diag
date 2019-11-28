package statement

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pingcap/tidb-foresight/wrapper/db"
	log "github.com/sirupsen/logrus"
)

type Statement struct {
	InstanceId       string `json:"instance_id" gorm:"column:INSTANCE_ID"`
	Node             string `json:"node" gorm:"column:NODE"`
	SchemaName       string `json:"schema" gorm:"column:SCHEMA_NAME"`
	SummaryBeginTime string `json:"-" gorm:"column:SUMMARY_BEGIN_TIME"`
	SummaryEndTime   string `json:"-" gorm:"column:SUMMARY_END_TIME"`
	StmtType         string `json:"type" gorm:"column:STMT_TYPE"`
	SumLatency       string `json:"sum_latency" gorm:"column:SUM_LATENCY"`
	AvgLatency       string `json:"avg_latency" gorm:"column:AVG_LATENCY"`
	ExecCount        string `json:"exec_count" gorm:"column:EXEC_COUNT"`
	AvgAffectedRows  string `json:"avg_affected_rows" gorm:"column:AVG_AFFECTED_ROWS"`
	AvgMem           string `json:"avg_mem" gorm:"column:AVG_MEM"`
	Digest           string `json:"digest" gorm:"column:DIGEST"`
	DigestText       string `json:"digest_text" gorm:"column:DIGEST_TEXT"`
	QuerySampleText  string `json:"query_sample_text" gorm:"column:QUERY_SAMPLE_TEXT`
	FirstSeen        string `json:"first_seen" gorm:"column:FIRST_SEEN`
	LastSeen         string `json:"last_seen" gorm:"column:LAST_SEEN`
}

type TiDBConnection struct {
	// The database connection
	db.DB

	// The ip:port pair of target TiDB endpoint
	instance string
}

func (m *statement) peekAnyConnection(instanceId string) (*TiDBConnection, error) {
	instance, err := m.inst.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}

	tidbs := strings.Split(instance.Tidb, ",")
	for _, tidb := range tidbs {
		if db, err := db.Open(db.MYSQL, fmt.Sprintf("root@(%s)/performance_schema", tidb)); err != nil {
			log.Warnf("open connection to tidb(%s) failed: %s", tidb, err)
			continue
		} else {
			return &TiDBConnection{db, tidb}, nil
		}
	}

	return nil, errors.New("open connection to all tidbs failed")
}

func (m *statement) peekAllConnection(instanceId string) ([]*TiDBConnection, error) {
	instance, err := m.inst.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}

	dbs := make([]*TiDBConnection, 0)

	tidbs := strings.Split(instance.Tidb, ",")
	for _, tidb := range tidbs {
		if db, err := db.Open(db.MYSQL, fmt.Sprintf("root@(%s)/performance_schema", tidb)); err != nil {
			for _, db := range dbs {
				db.Close()
			}
			return nil, err
		} else {
			dbs = append(dbs, &TiDBConnection{db, tidb})
		}
	}

	return dbs, nil
}

// Enable statements by set TiDB global variable
func (m *statement) EnableStatement(instanceId string) error {
	conn, err := m.peekAnyConnection(instanceId)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Raw("set global tidb_enable_stmt_summary = true;").Error()
}

// Disable statements by set TiDB global variable
func (m *statement) DisableStatement(instanceId string) error {
	conn, err := m.peekAnyConnection(instanceId)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Raw("set global tidb_enable_stmt_summary = false;").Error()
}

// Check if the statement is enabled in target cluster.
// The instanceId is the instance id of target cluster.
func (m *statement) IsStatementEnabled(instanceId string) (bool, error) {
	conn, err := m.peekAnyConnection(instanceId)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	res := struct {
		Enabled bool `gorm:"enabled"`
	}{}
	if err := conn.Raw("SELECT @@GLOBAL.tidb_enable_stmt_summary as enabled").Take(&res).Error(); err != nil {
		return false, err
	}
	return res.Enabled, nil
}

// How long will TiDB refresh statements table in seconds.
func (m *statement) GetRefreshInterval(instanceId string) (int, error) {
	conn, err := m.peekAnyConnection(instanceId)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	res := struct {
		Interval int `gorm:"duration"`
	}{}
	if err := conn.Raw("select @@GLOBAL.tidb_stmt_summary_refresh_interval as duration").Take(&res).Error(); err != nil {
		return 0, err
	}
	return res.Interval, nil
}

// Set how long the statements table should be dump to history.
func (m *statement) SetRefreshInterval(instanceId string, interval int) error {
	conn, err := m.peekAnyConnection(instanceId)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.Raw("set global tidb_stmt_summary_refresh_interval = ?", interval).Error()
}

// List statement of specified instance
func (m *statement) ListStatement(instanceId string, schemas []string, begin, end *time.Time) ([]*Statement, error) {
	stmts := []*Statement{}

	if err := m.db.Find(&stmts).Error(); err != nil {
		return nil, err
	}

	return stmts, nil
}

func (m *statement) ListStatementByDist(instanceId string, schemas []string, begin, end *time.Time) ([]*Statement, error) {
	stmts := []*Statement{}

	if err := m.db.Find(&stmts).Error(); err != nil {
		return nil, err
	}

	return stmts, nil
}
