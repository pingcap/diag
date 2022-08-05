package collector

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/sqltocsv"
	"github.com/joomcode/errorx"
	"github.com/pingcap/diag/pkg/models"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	// Register some standard stuff
	_ "github.com/pingcap/tidb/parser/test_driver"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/utils"
)

// PlanReplayerCollectorOptions collects sql explains and related statistics
type PlanReplayerCollectorOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	dbuser    string
	dbpasswd  string
	sqls      []string
	tables    map[table]struct{}
	resultDir string
	tlsCfg    *tls.Config
}

// Desc implements the Collector interface
func (c *PlanReplayerCollectorOptions) Desc() string {
	return "table statistics of components"
}

// GetBaseOptions implements the Collector interface
func (c *PlanReplayerCollectorOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *PlanReplayerCollectorOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations implements the Collector interface
func (c *PlanReplayerCollectorOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *PlanReplayerCollectorOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *PlanReplayerCollectorOptions) Prepare(_ *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *PlanReplayerCollectorOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	err := os.Mkdir(filepath.Join(c.resultDir, DirNameExplain), 0755)
	if err != nil {
		return err
	}
	err = os.Mkdir(filepath.Join(c.resultDir, DirStatistics), 0755)
	if err != nil {
		return err
	}
	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)
	tidbInstants := topo.TiDB
	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)

	t := task.NewBuilder(m.logger).
		Func(
			"collect explain sqls",
			func(ctx context.Context) error {
				db, sqldb := c.getDB(tidbInstants)
				if db == nil || sqldb == nil {
					return fmt.Errorf("cannot connect to any TiDB instance")
				}
				defer sqldb.Close()
				errs1 := c.CollectSqlExplain(sqldb)
				errs2 := c.CollectTableStats(ctx, db, sqldb)
				errs := append(errs1, errs2...)
				if len(errs) > 0 {
					// Only print warning is enough here
					logger.Warnf(strings.Join(errs, ";"))
				}
				return nil
			},
		).
		BuildAsStep("  - Querying Explain SQLs")

	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

func (c *PlanReplayerCollectorOptions) CollectTableStats(ctx context.Context, db *models.TiDBSpec, sqldb *sql.DB) []string {
	client := utils.NewHTTPClient(time.Second*15, c.tlsCfg)
	scheme := "http"
	if c.tlsCfg != nil {
		scheme = "https"
	}
	errs1 := c.collectTableStatistics(ctx, client, scheme, db)
	errs2 := c.collectTableStructures(sqldb)
	return append(errs1, errs2...)
}

func (c *PlanReplayerCollectorOptions) CollectSqlExplain(db *sql.DB) []string {
	var errs []string
	for index, sql := range c.sqls {
		rows, err := db.Query(fmt.Sprintf("explain %s", sql))
		if err != nil {
			errs = append(errs, fmt.Sprintf("sql:%v, err:%v", sql, err.Error()))
			continue
		}
		fileName := filepath.Join(c.resultDir, DirNameExplain, fmt.Sprintf("sql%d", index))
		_, err = os.Create(fileName)
		if err != nil {
			errs = append(errs, fmt.Sprintf("sql:%v, err:%v", sql, err.Error()))
			continue
		}
		err = sqltocsv.WriteFile(fileName, rows)
		if err != nil {
			errs = append(errs, fmt.Sprintf("sql:%v, err:%v", sql, err.Error()))
			continue
		}
	}
	return errs
}

func (c *PlanReplayerCollectorOptions) getDB(tidbInstants []*models.TiDBSpec) (*models.TiDBSpec, *sql.DB) {
	var sqldb *sql.DB
	var db *models.TiDBSpec
	for _, inst := range tidbInstants {
		cdb, err := func() (*sql.DB, error) {
			trydb, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/", c.dbuser, c.dbpasswd, inst.Host(), inst.MainPort()))
			if err != nil {
				return nil, err
			}
			err = trydb.Ping()
			if err != nil {
				defer trydb.Close()
				return nil, err
			}
			return trydb, nil
		}()
		if err != nil {
			continue
		}
		sqldb = cdb
		db = inst
	}
	return db, sqldb
}

func (c *PlanReplayerCollectorOptions) collectTableStatistics(ctx context.Context,
	client *utils.HTTPClient, scheme string, db *models.TiDBSpec) (errs []string) {
	for table := range c.tables {
		err := func() error {
			url := fmt.Sprintf("%s://%s/stats/dump/%s/%s", scheme, db.StatusURL(), table.dbName, table.tableName)
			response, err := client.Get(ctx, url)
			if err != nil {
				return fmt.Errorf("db:%v, table:%v, err:%v", table.dbName, table.tableName, err.Error())
			}
			path := filepath.Join(c.resultDir, DirStatistics, fmt.Sprintf("%s.%s.json", table.dbName, table.tableName))
			_, err = os.Create(path)
			if err != nil {
				return fmt.Errorf("db:%v, table:%v, err:%v", table.dbName, table.tableName, err.Error())
			}
			err = os.WriteFile(path, response, 0600)
			if err != nil {
				return fmt.Errorf("db:%v, table:%v, err:%v", table.dbName, table.tableName, err.Error())
			}
			return nil
		}()
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

func (c *PlanReplayerCollectorOptions) collectTableStructures(db *sql.DB) (errs []string) {
	defer db.Close()
	errs = c.collectTables()
	if errs != nil {
		return errs
	}
	for table := range c.tables {
		err := func() error {
			rows, err := db.Query(fmt.Sprintf("show create table %s.%s", table.dbName, table.tableName))
			if err != nil {
				return fmt.Errorf("db:%v, table:%v, err:%v", table.dbName, table.tableName, err.Error())
			}
			fileName := filepath.Join(c.resultDir, DirStatistics, fmt.Sprintf("%s.%s.schema", table.dbName, table.tableName))
			_, err = os.Create(fileName)
			if err != nil {
				return fmt.Errorf("db:%v, table:%v, err:%v", table.dbName, table.tableName, err.Error())
			}
			err = sqltocsv.WriteFile(fileName, rows)
			if err != nil {
				return fmt.Errorf("db:%v, table:%v, err:%v", table.dbName, table.tableName, err.Error())
			}
			return nil
		}()
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

func (c *PlanReplayerCollectorOptions) collectTables() []string {
	p := parser.New()
	var errs []string
	for _, sql := range c.sqls {
		stmtNodes, _, err := p.Parse(sql, "", "")
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		for _, stmtNode := range stmtNodes {
			v := tableVisitor{
				tables:   make(map[table]struct{}),
				cteNames: make(map[string]struct{}),
			}
			stmtNode.Accept(&v)
			for tbl := range v.tables {
				_, ok := v.cteNames[tbl.tableName]
				if !ok {
					c.tables[tbl] = struct{}{}
				}
			}
		}
	}
	return errs
}

type table struct {
	dbName    string
	tableName string
}

type tableVisitor struct {
	tables   map[table]struct{}
	cteNames map[string]struct{}
}

// Enter implements Visitor
func (v *tableVisitor) Enter(in ast.Node) (out ast.Node, skipChildren bool) {
	return in, false
}

// Leave implements Visitor
func (v *tableVisitor) Leave(in ast.Node) (out ast.Node, ok bool) {
	if t, ok := in.(*ast.TableName); ok {
		tbl := table{
			tableName: t.Name.L,
			dbName:    t.Schema.L,
		}
		v.tables[tbl] = struct{}{}
	} else if s, ok := in.(*ast.SelectStmt); ok {
		if s.With != nil && len(s.With.CTEs) > 0 {
			for _, cte := range s.With.CTEs {
				v.cteNames[cte.Name.L] = struct{}{}
			}
		}
	}
	return in, true
}
