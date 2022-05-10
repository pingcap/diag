package collector

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"github.com/joho/sqltocsv"
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
	"github.com/pingcap/tiup/pkg/utils"
)

type table struct {
	dbName    string
	tableName string
}

// StatisticsCollectorOptions collects tables statistics
type StatisticsCollectorOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	dbuser    string
	dbpasswd  string
	sqls      []string
	tables    []table
	resultDir string
	tlsCfg    *tls.Config
}

// Desc implements the Collector interface
func (c *StatisticsCollectorOptions) Desc() string {
	return "table statistics of components"
}

// GetBaseOptions implements the Collector interface
func (c *StatisticsCollectorOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *StatisticsCollectorOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations implements the Collector interface
func (c *StatisticsCollectorOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *StatisticsCollectorOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *StatisticsCollectorOptions) Prepare(_ *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *StatisticsCollectorOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	err := os.Mkdir(filepath.Join(c.resultDir, DirStatistics), 0755)
	if err != nil {
		return err
	}
	errs := c.collectTables()
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}

	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)

	tidbInstants := topo.TiDB

	t := task.NewBuilder(m.logger).
		Func(
			"collect table statistics",
			func(ctx context.Context) error {
				var db *models.TiDBSpec
				var sqldb *sql.DB
				for _, inst := range tidbInstants {
					err := func() error {
						trydb, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/", c.dbuser, c.dbpasswd, inst.Host(), inst.MainPort()))
						if err != nil {
							defer trydb.Close()
							return err
						}
						err = trydb.Ping()
						if err != nil {
							defer trydb.Close()
							return err
						}
						sqldb = trydb
						db = inst
						return nil
					}()
					if err == nil {
						break
					}
				}
				if db == nil || sqldb == nil {
					return fmt.Errorf("cannot connect to any TiDB instance")
				}
				client := utils.NewHTTPClient(time.Second*15, c.tlsCfg)
				scheme := "http"
				if c.tlsCfg != nil {
					scheme = "https"
				}
				errs := c.collectTableStatistics(ctx, client, scheme, db)
				if len(errs) > 0 {
					return fmt.Errorf(strings.Join(errs, "\n"))
				}
				errs = c.collectTableStructures(sqldb)
				if len(errs) > 0 {
					return fmt.Errorf(strings.Join(errs, "\n"))
				}
				return nil
			},
		).
		BuildAsStep("  - Querying Table Statistics")

	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

func (c *StatisticsCollectorOptions) collectTableStatistics(ctx context.Context,
	client *utils.HTTPClient, scheme string, db *models.TiDBSpec) (errs []string) {
	for _, table := range c.tables {
		err := func() error {
			url := fmt.Sprintf("%s://%s/stats/dump/%s/%s", scheme, db.StatusURL(), table.dbName, table.tableName)
			response, err := client.Get(ctx, url)
			if err != nil {
				return err
			}
			path := filepath.Join(c.resultDir, DirStatistics, fmt.Sprintf("%s.%s.json", table.dbName, table.tableName))
			_, err = os.Create(path)
			if err != nil {
				return err
			}
			return os.WriteFile(path, response, 0600)
		}()
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

func (c *StatisticsCollectorOptions) collectTableStructures(db *sql.DB) (errs []string) {
	defer db.Close()
	for _, table := range c.tables {
		err := func() error {
			rows, err := db.Query(fmt.Sprintf("show create table %s.%s", table.dbName, table.tableName))
			if err != nil {
				return err
			}
			fileName := filepath.Join(c.resultDir, DirStatistics, fmt.Sprintf("%s.%s.schema", table.dbName, table.tableName))
			_, err = os.Create(fileName)
			if err != nil {
				return err
			}
			return sqltocsv.WriteFile(fileName, rows)
		}()
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

func (c *StatisticsCollectorOptions) collectTables() []string {
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
				tables: make([]table, 0),
			}
			stmtNode.Accept(&v)
			c.tables = append(c.tables, v.tables...)
		}
	}
	return errs
}

type tableVisitor struct {
	tables []table
}

// Enter implements Visitor
func (v *tableVisitor) Enter(in ast.Node) (out ast.Node, skipChildren bool) {
	t, ok := in.(*ast.TableName)
	if ok {
		v.tables = append(v.tables, table{
			dbName:    t.Schema.L,
			tableName: t.Name.L,
		})
	}
	return in, false
}

// Leave implements Visitor
func (v *tableVisitor) Leave(in ast.Node) (out ast.Node, ok bool) {
	return in, true
}
