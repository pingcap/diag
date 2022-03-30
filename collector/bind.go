package collector

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/joho/sqltocsv"
	"github.com/joomcode/errorx"
	"github.com/pingcap/diag/pkg/models"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/task"
	"os"
	"path/filepath"
	"strings"
)

// BindCollectOptions are options used collecting component sql bind
type BindCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	dbuser    string
	dbpasswd  string
	resultDir string
	fileStats map[string][]CollectStat
}

type bindStruct struct {
	filename string
	sql      string
}

var collectedBind = []bindStruct{
	{
		"global_bind.csv",
		"select * from mysql.bind_info;",
	},
}

// Desc implements the Collector interface
func (c *BindCollectOptions) Desc() string {
	return "Bind of components"
}

// GetBaseOptions implements the Collector interface
func (c *BindCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *BindCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations implements the Collector interface
func (c *BindCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *BindCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *BindCollectOptions) Prepare(_ *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *BindCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	err := os.Mkdir(filepath.Join(c.resultDir, DirNameBind), 0755)
	if err != nil {
		return err
	}
	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)

	tidbInstants := topo.TiDB

	t := task.NewBuilder(m.logger).
		Func(
			"collect sql_bind",
			func(ctx context.Context) error {
				var db *sql.DB
				for _, inst := range tidbInstants {
					trydb, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/", c.dbuser, c.dbpasswd, inst.Host(), inst.MainPort()))
					defer trydb.Close()
					if err != nil {
						return err
					}
					err = trydb.Ping()
					if err == nil {
						db = trydb
						break
					}
				}
				if db == nil {
					return fmt.Errorf("cannot connect to any TiDB instance")
				}

				var errs []string
				for _, s := range collectedBind {
					rows, err := db.Query(s.sql)
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
					err = sqltocsv.WriteFile(filepath.Join(c.resultDir, DirNameBind, s.filename), rows)
					if err != nil {
						return err
					}
				}
				if len(errs) > 0 {
					return fmt.Errorf(strings.Join(errs, "\n"))
				}
				return nil
			},
		).
		BuildAsStep("  - Querying sql_bind")

	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}
