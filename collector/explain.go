package collector

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/sqltocsv"
	"github.com/joomcode/errorx"
	"github.com/pingcap/diag/pkg/models"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/task"
)

// ExplainCollectorOptions collects tables statistics
type ExplainCollectorOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	dbuser    string
	dbpasswd  string
	sqls      []string
	resultDir string
}

// Desc implements the Collector interface
func (c *ExplainCollectorOptions) Desc() string {
	return "table statistics of components"
}

// GetBaseOptions implements the Collector interface
func (c *ExplainCollectorOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *ExplainCollectorOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations implements the Collector interface
func (c *ExplainCollectorOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *ExplainCollectorOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *ExplainCollectorOptions) Prepare(_ *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *ExplainCollectorOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	err := os.Mkdir(filepath.Join(c.resultDir, DirNameExplain), 0755)
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
			"collect explain sqls",
			func(ctx context.Context) error {
				var db *sql.DB
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
					db = cdb
				}
				if db == nil {
					return fmt.Errorf("cannot connect to any TiDB instance")
				}
				defer db.Close()
				var errs []string
				for index, sql := range c.sqls {
					rows, err := db.Query(fmt.Sprintf("explain %s", sql))
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
					fileName := filepath.Join(c.resultDir, DirNameExplain, fmt.Sprintf("sql%d", index))
					_, err = os.Create(fileName)
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
					err = sqltocsv.WriteFile(filepath.Join(c.resultDir, DirNameExplain, fileName), rows)
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
				}
				if len(errs) > 0 {
					return fmt.Errorf(strings.Join(errs, "\n"))
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
