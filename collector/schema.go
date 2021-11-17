// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql" // import for sql
	"github.com/joho/sqltocsv"
	"github.com/joomcode/errorx"
	"github.com/pingcap/diag/pkg/models"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/task"
)

// SchemaCollectOptions are options used collecting component logs
type SchemaCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	dbuser    string
	dbpasswd  string
	resultDir string
	fileStats map[string][]CollectStat
}

type schemaStruct struct {
	filename string
	sql      string
}

var collectedSchemas = []schemaStruct{
	{
		"mysql.tidb.csv", `
select VARIABLE_NAME, VARIABLE_VALUE from mysql.tidb;`,
	},
}

// Desc implements the Collector interface
func (c *SchemaCollectOptions) Desc() string {
	return "Schema of components"
}

// GetBaseOptions implements the Collector interface
func (c *SchemaCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *SchemaCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *SchemaCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *SchemaCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *SchemaCollectOptions) Prepare(_ *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *SchemaCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	err := os.Mkdir(filepath.Join(c.resultDir, DirNameSchema), 0755)
	if err != nil {
		return err
	}
	ctx := ctxt.New(context.Background(), c.opt.Concurrency)

	tidbInstants := topo.TiDB

	t := task.NewBuilder(m.DisplayMode).
		Func(
			"collect db_vars",
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
				for _, s := range collectedSchemas {
					rows, err := db.Query(s.sql)
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
					err = sqltocsv.WriteFile(filepath.Join(c.resultDir, DirNameSchema, s.filename), rows)
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
		BuildAsStep("  - Querying db_vars")

	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}
