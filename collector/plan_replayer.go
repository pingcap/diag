package collector

import (
	"archive/zip"
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	return "collect plan replayer"
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
	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)
	tidbInstants := topo.TiDB
	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)

	t := task.NewBuilder(m.logger).
		Func(
			"collect plan replayer",
			func(ctx context.Context) error {
				db, sqldb := c.getDB(tidbInstants)
				if db == nil || sqldb == nil {
					return fmt.Errorf("cannot connect to any TiDB instance")
				}
				defer sqldb.Close()
				err := c.collectPlanReplayer(ctx, sqldb, db)
				if err != nil {
					logger.Warnf("error happened during plan replayer, err:%v", err)
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

func (c *PlanReplayerCollectorOptions) collectPlanReplayer(ctx context.Context, db *sql.DB, dbInstance *models.TiDBSpec) error {
	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)
	zf, err := os.Create(filepath.Join(c.resultDir, "plan_replayer.zip"))
	if err != nil {
		return err
	}
	// Create zip writer
	zw := zip.NewWriter(zf)
	defer func() {
		err = zw.Close()
		if err != nil {
			logger.Warnf("Closing zip writer failed, err:%v", err)
		}
		err = zf.Close()
		if err != nil {
			logger.Warnf("Closing zip file failed, err:%v", err)
		}
	}()
	errs := c.collectTableStructures(zw, db)
	if len(errs) > 0 {
		logger.Warnf("error happened during collect table schema, err:%v", strings.Join(errs, ","))
	}
	errs = c.collectTableStats(ctx, zw, dbInstance)
	if len(errs) > 0 {
		logger.Warnf("error happened during collect table stats, err:%v", strings.Join(errs, ","))
	}
	errs = c.collectSQLExplain(zw, db)
	if len(errs) > 0 {
		logger.Warnf("error happened during explain sql, err:%v", strings.Join(errs, ","))
	}
	return nil
}

/*
 |-meta.txt
 |-schema
 |	 |-db1.table1.schema.txt
 |	 |-db2.table2.schema.txt
 |	 |-....
*/
func (c *PlanReplayerCollectorOptions) collectTableStructures(zw *zip.Writer, db *sql.DB) (errs []string) {
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
			var tableName string
			var showCreate string
			for rows.Next() {
				err := rows.Scan(&tableName, &showCreate)
				if err != nil {
					return fmt.Errorf("show create table %s.%s failed, err:%v", table.dbName, table.tableName, err)
				}
			}
			defer rows.Close()
			fw, err := zw.Create(fmt.Sprintf("schema/%s.%s.schema.txt", table.dbName, table.tableName))
			if err != nil {
				return fmt.Errorf("generate schema/%s.%s.schema.txt failed, err:%v", table.dbName, table.tableName, err)
			}
			fmt.Fprintf(fw, "create database if not exists `%v`; use `%v`;", table.dbName, table.dbName)
			fmt.Fprintf(fw, "%s", showCreate)
			return nil
		}()
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

/*
 |-meta.txt
 |-stats
 |   |-stats1.json
 |   |-stats2.json
 |   |-....
*/
func (c *PlanReplayerCollectorOptions) collectTableStats(ctx context.Context, zw *zip.Writer,
	db *models.TiDBSpec) (errs []string) {
	client := utils.NewHTTPClient(time.Second*15, c.tlsCfg)
	scheme := "http"
	if c.tlsCfg != nil {
		scheme = "https"
	}
	for table := range c.tables {
		err := func() error {
			url := fmt.Sprintf("%s://%s/stats/dump/%s/%s", scheme, db.StatusURL(), table.dbName, table.tableName)
			response, err := client.Get(ctx, url)
			if err != nil {
				return fmt.Errorf("dump %s.%s failed, err:%v", table.dbName, table.tableName, err.Error())
			}
			fw, err := zw.Create(fmt.Sprintf("stats/%s.%s.json", table.dbName, table.tableName))
			if err != nil {
				return fmt.Errorf("create stats/%s.%s.json failed, err:%v", table.dbName, table.tableName, err)
			}
			_, err = fw.Write(response)
			if err != nil {
				return fmt.Errorf("write %s.%s stats failed, err:%v", table.dbName, table.tableName, err)
			}
			return nil
		}()
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

/*
 |-sqls.sql
 |_explain
     |-explain1.txt
	 |-explain2.txt
 	 |-....
*/
func (c *PlanReplayerCollectorOptions) collectSQLExplain(zw *zip.Writer, db *sql.DB) (errs []string) {
	for index, sql := range c.sqls {
		err := func() error {
			fw, err := zw.Create(fmt.Sprintf("explain/explain%v.txt", index))
			if err != nil {
				return fmt.Errorf("create explain sql failed, err:%v", err)
			}
			rows, err := db.Query(fmt.Sprintf("explain %s", sql))
			if err != nil {
				return fmt.Errorf("explain sql:%s failed, err:%v", sql, err)
			}
			defer rows.Close()
			cols, err := rows.Columns()
			if err != nil {
				return fmt.Errorf("explain sql:%s failed, err:%v", sql, err)
			}
			buff := make([]interface{}, len(cols))
			data := make([]string, len(cols))
			for i, _ := range buff {
				buff[i] = &data[i]
			}
			for rows.Next() {
				err := rows.Scan(buff...)
				if err != nil {
					return fmt.Errorf("explain sql:%s failed, err:%v", sql, err)
				}
				fmt.Fprintf(fw, "%s\n", strings.Join(data, "\t"))
			}
			return nil
		}()
		if err != nil {
			errs = append(errs, err.Error())
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
