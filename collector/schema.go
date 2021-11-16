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

type infoSchema struct {
	filename string
	sql      string
}

var infoSchemas []infoSchema = []infoSchema{
	{
		"avg_process_time_by_plan.csv", `
SELECT a.Digest, a.Plan_Digest, a.avg_process_time, a.last_time FROM
(
SELECT Digest, Plan_Digest, AVG(Process_time) as avg_process_time, MAX('Time') as last_time FROM slow_query where Time BETWEEN DATE_SUB(CURRENT_TIMESTAMP(6), INTERVAL 7 DAY) AND CURRENT_TIMESTAMP(6) AND Process_keys>1000000 GROUP BY Digest,Plan_Digest
) a join
(
SELECT Digest, COUNT(distinct(Plan_Digest)) as plan_count FROM slow_query where Time BETWEEN DATE_SUB(CURRENT_TIMESTAMP(6), INTERVAL 7 DAY) AND CURRENT_TIMESTAMP(6) AND Process_keys>1000000 GROUP BY Digest having plan_count > 1
) b on a.Digest = b.Digest;`,
	},
	{
		"key_old_version_plan.csv", `
select distinct Digest, Plan_Digest
FROM slow_query 
WHERE process_keys > 10000 AND total_keys = 2 * process_keys and time > DATE_SUB(CURDATE(), INTERVAL 7 DAY)`,
	},
	{
		"mysql.tidb.csv", `
select VARIABLE_NAME, VARIABLE_VALUE from mysql.tidb;`,
	},
	{
		"skip_toomany_keys_plan.csv", `
SELECT distinct Digest, Plan_Digest FROM slow_query WHERE ABS(Rocksdb_delete_skipped_count-Process_keys) <= 1000 AND Process_keys > 10000 AND Time > DATE_SUB(CURRENT_TIMESTAMP(6), INTERVAL 7 DAY);`,
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
	err := os.Mkdir(filepath.Join(c.resultDir, "info_schema"), 0755)
	if err != nil {
		return err
	}
	ctx := ctxt.New(context.Background(), c.opt.Concurrency)

	tidbInstants := topo.TiDB

	t := task.NewBuilder(m.DisplayMode).
		Func(
			"collect info_schema",
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

				_, err = db.Exec("USE information_schema;")
				if err != nil {
					return err
				}

				var errs []string
				for _, s := range infoSchemas {
					rows, err := db.Query(s.sql)
					if err != nil {
						errs = append(errs, err.Error())
						continue
					}
					err = sqltocsv.WriteFile(filepath.Join(c.resultDir, "info_schema", s.filename), rows)
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
		BuildAsStep("  - Querying infoSchema")

	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}
