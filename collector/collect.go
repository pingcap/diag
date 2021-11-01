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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/pingcap/diag/pkg/utils"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/tui"
)

// types of data to collect
const (
	CollectTypeSystem  = "system"
	CollectTypeMonitor = "monitor"
	CollectTypeLog     = "log"
	CollectTypeConfig  = "config"
)

// Collector is the configuration defining an collecting job
type Collector interface {
	Prepare(*Manager, *spec.Specification) (map[string][]CollectStat, error)
	Collect(*Manager, *spec.Specification) error
	GetBaseOptions() *BaseOptions
	SetBaseOptions(*BaseOptions)
	Desc() string // a brief self description
}

// BaseOptions contains the options for check command
type BaseOptions struct {
	Cluster     string                  // cluster name
	User        string                  // username to login to the SSH server
	UsePassword bool                    // use password instead of identity file for ssh connection
	SSH         *tui.SSHConnectionProps // SSH credentials
	ScrapeBegin string                  // start timepoint when collecting metrics and logs
	ScrapeEnd   string                  // stop timepoint when collecting metrics and logs
}

// CollectOptions contains the options defining which type of data to collect
type CollectOptions struct {
	Include         set.StringSet
	Exclude         set.StringSet
	MetricsFilter   []string
	Dir             string // target directory to store collected data
	Limit           int    // rate limit of SCP
	CompressMetrics bool   // compress of files during collecting
	CompressLogs    bool   // compress of files during collecting
	ExitOnError     bool   // break the process and exit when an error occurs
}

// CollectStat is estimated size stats of data to be collected
type CollectStat struct {
	Target string
	Size   int64
}

// CollectClusterInfo collects information and metrics from a tidb cluster
func (m *Manager) CollectClusterInfo(
	opt *BaseOptions,
	cOpt *CollectOptions,
	gOpt *operator.Options,
) error {
	var topo spec.Specification

	exist, err := m.specManager.Exist(opt.Cluster)
	if err != nil {
		return err
	}
	if !exist {
		return perrs.Errorf("cluster %s does not exist", opt.Cluster)
	}

	metadata, err := spec.ClusterMetadata(opt.Cluster)
	if err != nil {
		return err
	}
	opt.User = metadata.User
	opt.SSH.IdentityFile = m.specManager.Path(opt.Cluster, "ssh", "id_rsa")
	topo = *metadata.Topology

	// parse time range
	end, err := utils.ParseTime(opt.ScrapeEnd)
	if err != nil {
		return err
	}
	// if the begin time point is a minus integer, assume it as hour offset
	var start time.Time
	if offset, err := strconv.Atoi(opt.ScrapeBegin); err == nil && offset < 0 {
		start = end.Add(time.Hour * time.Duration(offset))
		// update time string in setting to ensure all collectors work properly
		opt.ScrapeBegin = start.Format(time.RFC3339)
	} else {
		start, err = utils.ParseTime(opt.ScrapeBegin)
		if err != nil {
			return err
		}
	}

	resultDir, err := m.getOutputDir(cOpt.Dir)
	if err != nil {
		return err
	}

	collectorSet := map[string]bool{
		CollectTypeSystem:  false,
		CollectTypeMonitor: false,
		CollectTypeLog:     false,
		CollectTypeConfig:  false,
	}
	for name := range collectorSet {
		if canCollect(cOpt, name) {
			collectorSet[name] = true
		}
	}

	// build collector list
	collectors := []Collector{
		&MetaCollectOptions{ // cluster metadata, always collected
			BaseOptions: opt,
			opt:         gOpt,
			session:     m.session,
			collectors:  collectorSet,
			resultDir:   resultDir,
			filePath:    m.specManager.Path(opt.Cluster, "meta.yaml"),
		},
	}

	// collect data from monitoring system
	if canCollect(cOpt, CollectTypeMonitor) {
		collectors = append(collectors,
			&AlertCollectOptions{ // alerts
				BaseOptions: opt,
				opt:         gOpt,
				resultDir:   resultDir,
				compress:    cOpt.CompressMetrics,
			},
			&MetricCollectOptions{ // metrics
				BaseOptions: opt,
				opt:         gOpt,
				resultDir:   resultDir,
				compress:    cOpt.CompressMetrics,
				filter:      cOpt.MetricsFilter,
			},
		)
	}

	// populate SSH credentials if needed
	if canCollect(cOpt, CollectTypeSystem) ||
		canCollect(cOpt, CollectTypeLog) ||
		canCollect(cOpt, CollectTypeConfig) {
		// collect data from remote servers
		var sshConnProps *tui.SSHConnectionProps = &tui.SSHConnectionProps{}
		if gOpt.SSHType != executor.SSHTypeNone {
			var err error
			if sshConnProps, err = tui.ReadIdentityFileOrPassword(opt.SSH.IdentityFile, opt.UsePassword); err != nil {
				return err
			}
		}
		opt.SSH = sshConnProps
	}

	if canCollect(cOpt, CollectTypeSystem) {
		collectors = append(collectors, &SystemCollectOptions{
			BaseOptions: opt,
			opt:         gOpt,
			resultDir:   resultDir,
		})
	}

	// collect log files
	if canCollect(cOpt, CollectTypeLog) {
		collectors = append(collectors,
			&LogCollectOptions{
				BaseOptions: opt,
				opt:         gOpt,
				limit:       cOpt.Limit,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
			})
	}

	// collect config files
	if canCollect(cOpt, CollectTypeConfig) {
		collectors = append(collectors,
			&ConfigCollectOptions{
				BaseOptions: opt,
				opt:         gOpt,
				limit:       cOpt.Limit,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
			})
	}

	// prepare
	// run collectors
	prepareErrs := make(map[string]error)
	stats := make([]map[string][]CollectStat, 0)
	for _, c := range collectors {
		fmt.Printf("Collecting %s...\n", c.Desc())
		stat, err := c.Prepare(m, &topo)
		if err != nil {
			if cOpt.ExitOnError {
				return err
			}
			fmt.Println(color.YellowString("Error collecting %s: %s, the data might be incomplete.", c.Desc(), err))
			prepareErrs[c.Desc()] = err
		}
		stats = append(stats, stat)
	}

	// show time range
	fmt.Printf(`Time range:
  %s - %s (Local)
  %s - %s (UTC)
  (total %.0f seconds)
`,
		color.HiYellowString(start.Local().Format(time.RFC3339)), color.HiYellowString(end.Local().Format(time.RFC3339)),
		color.HiYellowString(start.UTC().Format(time.RFC3339)), color.HiYellowString(end.UTC().Format(time.RFC3339)),
		end.Sub(start).Seconds(),
	)

	// confirm before really collect
	if err := confirmStats(stats, resultDir); err != nil {
		return err
	}
	err = os.MkdirAll(resultDir, 0755)
	if err != nil {
		return err
	}

	// run collectors
	collectErrs := make(map[string]error)
	for _, c := range collectors {
		fmt.Printf("Collecting %s...\n", c.Desc())
		if err := c.Collect(m, &topo); err != nil {
			if cOpt.ExitOnError {
				return err
			}
			fmt.Println(color.YellowString("Error collecting %s: %s, the data might be incomplete.", c.Desc(), err))
			collectErrs[c.Desc()] = err
		}
	}

	if len(collectErrs) > 0 {
		fmt.Println(color.RedString("Some errors occured during the process, please check if data needed are complete:"))
		for k, v := range prepareErrs {
			fmt.Printf("%s:\t%s\n", k, v)
		}
		for k, v := range collectErrs {
			fmt.Printf("%s:\t%s\n", k, v)
		}
	}
	fmt.Printf("Collected data are stored in %s\n", color.CyanString(resultDir))
	return nil
}

// prepare output dir of collected data
func (m *Manager) getOutputDir(dir string) (string, error) {
	if dir == "" {
		dir = filepath.Join(".", "diag-"+m.session)
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return dir, err
	}

	dirInfo, err := os.Stat(dir)
	// need mkdir if output dir not exists
	if err != nil {
		return dir, nil
	}

	if dirInfo.IsDir() {
		readdir, err := os.ReadDir(dir)
		if err != nil {
			return dir, err
		}
		if len(readdir) == 0 {
			return dir, nil
		}
		return dir, fmt.Errorf("%s is not an empty directory", dir)
	}

	return dir, fmt.Errorf("%s is not a directory", dir)
}

func confirmStats(stats []map[string][]CollectStat, resultDir string) error {
	fmt.Printf("Estimated size of data to collect:\n")
	var total int64
	statTable := [][]string{{"Host", "Size", "Target"}}
	for _, stat := range stats {
		if stat == nil {
			continue
		}
		for host, items := range stat {
			if len(items) < 1 {
				continue
			}
			for _, s := range items {
				total += s.Size
				statTable = append(statTable, []string{host, color.CyanString(readableSize(s.Size)), s.Target})
			}
		}
	}
	statTable = append(statTable, []string{"Total", color.YellowString(readableSize(total)), "(inaccurate)"})
	tui.PrintTable(statTable, true)
	fmt.Printf("These data will be stored in %s\n", color.CyanString(resultDir))
	return tui.PromptForConfirmOrAbortError("Do you want to continue? [y/N]: ")
}

func readableSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func canCollect(cOpt *CollectOptions, cType string) bool {
	return cOpt.Include.Exist(cType) && !cOpt.Exclude.Exist(cType)
}
