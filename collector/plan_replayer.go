package collector

import (
	"archive/zip"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
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
	opt            *operator.Options // global operations from cli
	dbuser         string
	dbpasswd       string
	sqls           []string
	views          map[table]struct{}
	tables         map[table]struct{}
	tablesAndViews map[table]struct{}
	resultDir      string
	tlsCfg         *tls.Config
	currDB         string
}

// Desc implements the Collector interface
func (c *PlanReplayerCollectorOptions) Desc() string {
	return "collect information for plan replayer"
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
	tidbInstances := topo.TiDB
	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)

	t := task.NewBuilder(m.logger).
		Func(
			"collect information for plan replayer",
			func(ctx context.Context) error {
				db, sqldb := c.getDBFromTopo(tidbInstances)
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
	errs := c.collectTables()
	if errs != nil {
		return fmt.Errorf("collect table failed, errs:%s", strings.Join(errs, ","))
	}
	errs = c.separateTableAndView(ctx, dbInstance, c.tablesAndViews)
	if errs != nil {
		return fmt.Errorf("separate tables and views failed, err:%s", strings.Join(errs, ","))
	}
	errs = c.collectTableStructures(zw, db)
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
	errs = c.collectDBVars(zw, db)
	if len(errs) > 0 {
		logger.Warnf("error happened during collect dbvars, err:%v", strings.Join(errs, ","))
	}
	errs = c.collectTiflashReplicas(zw, db)
	if len(errs) > 0 {
		logger.Warnf("error happened during collect tiflash replica info, err:%v", strings.Join(errs, ","))
	}
	return nil
}

/*
|-table_tiflash_replica.txt
*/
func (c *PlanReplayerCollectorOptions) collectTiflashReplicas(zw *zip.Writer, db *sql.DB) (errs []string) {
	vf, err := zw.Create("table_tiflash_replica.txt")
	if err != nil {
		errs = append(errs, fmt.Errorf("create table_tiflash_replica.txt failed, err:%v", err).Error())
		return errs
	}
	// nolint: gosec
	sql := "SELECT TABLE_SCHEMA,TABLE_NAME,REPLICA_COUNT FROM INFORMATION_SCHEMA.TIFLASH_REPLICA WHERE TABLE_SCHEMA= ? AND TABLE_NAME = ? AND REPLICA_COUNT >0"
	for table := range c.tables {
		err := func() error {
			rows, err := db.Query(sql, table.dbName, table.tableName)
			if err != nil {
				return fmt.Errorf("failed to query tiflash replicas, err:%v", err)
			}
			for rows.Next() {
				var dbName, tableName, count string
				if err := rows.Scan(&dbName, &tableName, &count); err != nil {
					return err
				}
				r := []string{
					dbName, tableName, count,
				}
				fmt.Fprintf(vf, "%s\n", strings.Join(r, "\t"))
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
|-variables.toml
*/
func (c *PlanReplayerCollectorOptions) collectDBVars(zw *zip.Writer, db *sql.DB) (errs []string) {
	err := func() error {
		vf, err := zw.Create("variables.toml")
		if err != nil {
			return fmt.Errorf("create variables.toml failed, err:%v", err)
		}
		rows, err := db.Query("show variables")
		if err != nil {
			return fmt.Errorf("show variables failed, err:%v", err)
		}
		defer rows.Close()
		varMap := make(map[string]string)
		var name, value string
		for rows.Next() {
			err := rows.Scan(&name, &value)
			if err != nil {
				return fmt.Errorf("show variables failed, err:%v", err)
			}
			varMap[name] = value
		}
		if err := toml.NewEncoder(vf).Encode(varMap); err != nil {
			return fmt.Errorf("dump varMap failed, err:%v", err)
		}
		return nil
	}()
	if err != nil {
		errs = append(errs, err.Error())
	}
	return errs
}

/*
|-meta.txt
|-schema
|	 |-db1.table1.schema.txt
|	 |-db2.table2.schema.txt
|	 |-....
|-view
| 	 |-db1.view1.view.txt
|	 |-db2.view2.view.txt
|	 |-....
*/
func (c *PlanReplayerCollectorOptions) collectTableStructures(zw *zip.Writer, db *sql.DB) (errs []string) {
	createTblOrView := func(isTable bool, dbName, name string) error {
		rows, err := db.Query(fmt.Sprintf("show create table %s.%s", dbName, name))
		if err != nil {
			return fmt.Errorf("db:%v, table:%v, err:%v", dbName, name, err.Error())
		}
		defer rows.Close()
		var showCreate string
		if isTable {
			var tableName string
			for rows.Next() {
				err := rows.Scan(&tableName, &showCreate)
				if err != nil {
					return fmt.Errorf("show create table %s.%s failed, err:%v", dbName, name, err)
				}
			}
		} else {
			var viewName, character, collation string
			for rows.Next() {
				err := rows.Scan(&viewName, &showCreate, &character, &collation)
				if err != nil {
					return fmt.Errorf("show create table %s.%s failed, err:%v", dbName, name, err)
				}
			}
		}
		var fw io.Writer
		if isTable {
			fw, err = zw.Create(fmt.Sprintf("schema/%s.%s.schema.txt", dbName, name))
			if err != nil {
				return fmt.Errorf("generate schema/%s.%s.schema.txt failed, err:%v", dbName, name, err)
			}
		} else {
			fw, err = zw.Create(fmt.Sprintf("view/%s.%s.view.txt", dbName, name))
			if err != nil {
				return fmt.Errorf("generate view/%s.%s.view.txt failed, err:%v", dbName, name, err)
			}
		}
		fmt.Fprintf(fw, "create database if not exists `%v`; use `%v`;", dbName, dbName)
		fmt.Fprintf(fw, "%s", showCreate)
		return nil
	}

	for table := range c.tables {
		err := createTblOrView(true, table.dbName, table.tableName)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	for view := range c.views {
		err := createTblOrView(false, view.dbName, view.tableName)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

type SchemaResponse struct {
	View *ViewResponse `json:"view"`
}

type ViewResponse struct {
	ViewSelect string `json:"view_select"`
}

func (c *PlanReplayerCollectorOptions) separateTableAndView(ctx context.Context, db *models.TiDBSpec, tableAndViews map[table]struct{}) (errs []string) {
	client := utils.NewHTTPClient(time.Second*15, c.tlsCfg)
	scheme := "http"
	if c.tlsCfg != nil {
		scheme = "https"
	}
	for table := range tableAndViews {
		err := func() error {
			url := fmt.Sprintf("%s://%s/schema/%s/%s", scheme, db.StatusURL(), table.dbName, table.tableName)
			b, err := client.Get(ctx, url)
			if err != nil {
				return err
			}
			r := &SchemaResponse{}
			err = json.Unmarshal(b, r)
			if err != nil {
				return err
			}
			if r.View == nil {
				c.tables[table] = struct{}{}
			} else {
				c.views[table] = struct{}{}
				subTablesAndViews, err := extractTablesAndViews(c.currDB, r.View.ViewSelect)
				if err != nil {
					return err
				}
				subErrs := c.separateTableAndView(ctx, db, subTablesAndViews)
				if len(subErrs) > 0 {
					errs = append(errs, subErrs...)
				}
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
			for i := range buff {
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

func (c *PlanReplayerCollectorOptions) getDB(inst *models.TiDBSpec) (*sql.DB, error) {
	trydb, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		c.dbuser, c.dbpasswd, inst.Host(), inst.MainPort(), c.currDB))
	if err != nil {
		return nil, err
	}
	err = trydb.Ping()
	if err != nil {
		defer trydb.Close()
		return nil, err
	}
	return trydb, nil
}

func (c *PlanReplayerCollectorOptions) getDBFromTopo(tidbInstants []*models.TiDBSpec) (*models.TiDBSpec, *sql.DB) {
	var sqldb *sql.DB
	var db *models.TiDBSpec
	for _, inst := range tidbInstants {
		cdb, err := c.getDB(inst)
		if err != nil {
			continue
		}
		sqldb = cdb
		db = inst
	}
	return db, sqldb
}

func (c *PlanReplayerCollectorOptions) collectTables() (errs []string) {
	for _, sql := range c.sqls {
		err := func() error {
			r, err := extractTablesAndViews(c.currDB, sql)
			if err != nil {
				return err
			}
			for table := range r {
				c.tablesAndViews[table] = struct{}{}
			}
			return nil
		}()
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

type table struct {
	dbName    string
	tableName string
}

type tableVisitor struct {
	currDB   string
	tables   map[table]struct{}
	cteNames map[string]struct{}
}

func newTableVisitor(currDB string) *tableVisitor {
	v := &tableVisitor{
		currDB:   currDB,
		tables:   make(map[table]struct{}),
		cteNames: make(map[string]struct{}),
	}
	return v
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
		if len(t.Schema.L) < 1 {
			tbl.dbName = v.currDB
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

func extractTablesAndViews(currDB, sql string) (map[table]struct{}, error) {
	p := parser.New()
	stmtNodes, _, err := p.Parse(sql, "", "")
	if err != nil {
		return nil, err
	}
	tablesAndViews := make(map[table]struct{})
	for _, stmtNode := range stmtNodes {
		v := newTableVisitor(currDB)
		stmtNode.Accept(v)
		for tbl := range v.tables {
			_, ok := v.cteNames[tbl.tableName]
			if !ok {
				tablesAndViews[tbl] = struct{}{}
			}
		}
	}
	return tablesAndViews, nil
}
