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
	"archive/zip"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joomcode/errorx"
	"github.com/pingcap/diag/pkg/models"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/utils"
)

// PerfCollectOptions are options used collecting pref info
type PerfCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	limit     int
	duration  int // scp rate limit
	resultDir string
	fileStats map[string][]CollectStat
	compress  bool
}

// Desc implements the Collector interface
func (c *PerfCollectOptions) Desc() string {
	return "Pref info of components"
}

// GetBaseOptions implements the Collector interface
func (c *PerfCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *PerfCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *PerfCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *PerfCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *PerfCollectOptions) Prepare(_ *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *PerfCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)

	collectePerfTasks := []*task.StepDisplay{}

	for _, inst := range topo.TiDB {
		collectePerfTasks = append(collectePerfTasks, buildPerfCollectingWithTiDBTasks(ctx, inst, c.resultDir, c.duration, nil))
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Query profile info", false, collectePerfTasks...).Build()

	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}

// perfInfo  profile information
type perfInfo struct {
	filename string
	url      string
	zip      bool
}

// TaskRawDataType, error) {
// var profilingRawDataType TaskRawDataType
// var fileExtenstion string
// secs := strconv.Itoa(int(duration))
// var url string
// switch profilingType {
// case ProfilingTypeCPU:
// 	url = "/debug/pprof/profile?seconds=" + secs
// 	profilingRawDataType = RawDataTypeProtobuf
// 	fileExtenstion = "*.proto"
// case ProfilingTypeHeap:
// 	url = "/debug/pprof/heap"
// 	profilingRawDataType = RawDataTypeProtobuf
// 	fileExtenstion = "*.proto"
// case ProfilingTypeGoroutine:
// 	url = "/debug/pprof/goroutine?debug=1"
// 	profilingRawDataType = RawDataTypeText
// 	fileExtenstion = "*.txt"
// case ProfilingTypeMutex:
// 	url = "/debug/pprof/mutex?debug=1"
// 	profilingRawDataType = RawDataTypeText
// 	fileExtenstion = "*.txt"
// }

// http://172.16.7.147:2379

func buildPerfCollectingWithTiDBTasks(ctx context.Context, inst *models.TiDBSpec, resultDir string, duration int, tlsCfg *tls.Config) *task.StepDisplay {
	var perfInfos []perfInfo
	scheme := "http"
	if tlsCfg != nil {
		scheme = "https"
	}

	host := inst.Host()
	instDir, ok := inst.Attributes()["deploy_dir"].(string)
	if !ok {
		// for tidb-operator deployed cluster
		instDir = ""
	}
	if pod, ok := inst.Attributes()["pod"].(string); ok {
		host = pod
	}

	switch inst.Type() {
	case models.ComponentTypeTiDB:
		url := fmt.Sprintf("%s:%d/debug/zip?seconds=%d", host, inst.StatusPort(), duration)
		perfInfos = append(perfInfos, perfInfo{"debug.zip", url, true})
	default:
		// not supported yet, just ignore
		return nil
	}

	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)
	t := task.NewBuilder(logger).
		Func(
			fmt.Sprintf("querying %s:%d", host, inst.StatusPort()),
			func(ctx context.Context) error {
				c := utils.NewHTTPClient(time.Second*time.Duration(duration+3), tlsCfg)
				for _, perfInfo := range perfInfos {
					url := fmt.Sprintf("%s://%s", scheme, perfInfo.url)
					resp, err := c.Get(ctx, url)
					if err != nil {
						return fmt.Errorf("fail querying %s: %s, continue", url, err)
					}

					fpath := filepath.Join(resultDir, host, instDir, "perf")
					if err := utils.CreateDir(fpath); err != nil {
						return err
					}

					fFile := filepath.Join(resultDir, host, instDir, "perf", perfInfo.filename)
					err = os.WriteFile(
						fFile,
						resp,
						0644,
					)
					if err != nil {
						return err
					}

					if perfInfo.zip {
						Unzip(fFile, fpath)
						os.Remove(fFile)                           // delete zip file
						os.Remove(filepath.Join(fpath, "version")) // delete version duplicate file
						os.Remove(filepath.Join(fpath, "config"))  // delete version duplicate file
					}

				}
				return nil
			},
		).
		BuildAsStep(fmt.Sprintf(
			"  - Querying configs for %s %s:%d",
			inst.Type(),
			inst.Host(),
			inst.MainPort(),
		))

	return t
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}
