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

	"github.com/fatih/color"
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
	Prepare(*spec.Specification) (map[string][]CollectStat, error)
	Collect(*spec.Specification) error
	GetBaseOptions() *BaseOptions
	SetBaseOptions(*BaseOptions)
	Desc() string // a brief self description
}

// BaseOptions contains the options for check command
type BaseOptions struct {
	User        string                  // username to login to the SSH server
	UsePassword bool                    // use password instead of identity file for ssh connection
	SSH         *tui.SSHConnectionProps // SSH credentials
	ScrapeBegin string                  // start timepoint when collecting metrics and logs
	ScrapeEnd   string                  // stop timepoint when collecting metrics and logs
}

// CollectOptions contains the options defining which type of data to collect
type CollectOptions struct {
	Include set.StringSet
	Exclude set.StringSet
	Dir     string // target directory to store collected data
	Limit   int    // rate limit of SCP
}

// CollectStat is estimated size stats of data to be collected
type CollectStat struct {
	Target string
	Size   int64
}

// CollectClusterInfo collects information and metrics from a tidb cluster
func (m *Manager) CollectClusterInfo(
	clusterName string,
	opt *BaseOptions,
	cOpt *CollectOptions,
	gOpt *operator.Options,
) error {
	var topo spec.Specification

	exist, err := m.specManager.Exist(clusterName)
	if err != nil {
		return err
	}
	if !exist {
		return perrs.Errorf("cluster %s does not exist", clusterName)
	}

	metadata, err := spec.ClusterMetadata(clusterName)
	if err != nil {
		return err
	}
	opt.User = metadata.User
	opt.SSH.IdentityFile = m.specManager.Path(clusterName, "ssh", "id_rsa")
	topo = *metadata.Topology

	// prepare output dir of collected data
	var resultDir string
	if cOpt.Dir == "" {
		resultDir = m.specManager.Path(clusterName, "collector", m.session)
	} else {
		fp, err := filepath.Abs(cOpt.Dir)
		if err != nil {
			return err
		}
		resultDir = filepath.Join(fp, m.session)
	}
	if err := os.MkdirAll(resultDir, 0755); err != nil {
		return err
	}

	// build collector list
	collectors := []Collector{
		&MetaCollectOptions{ // cluster metadata, always collected
			BaseOptions: opt,
			opt:         gOpt,
			resultDir:   resultDir,
			filePath:    m.specManager.Path(clusterName, "meta.yaml"),
		},
	}

	// collect data from monitoring system
	if canCollect(cOpt, CollectTypeMonitor) {
		collectors = append(collectors,
			&AlertCollectOptions{ // alerts
				BaseOptions: opt,
				opt:         gOpt,
				resultDir:   resultDir,
			},
			&MetricCollectOptions{ // metrics
				BaseOptions: opt,
				opt:         gOpt,
				resultDir:   resultDir,
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
	stats := make([]map[string][]CollectStat, 0)
	for _, c := range collectors {
		fmt.Printf("Collecting %s...\n", c.Desc())
		stat, err := c.Prepare(&topo)
		if err != nil {
			return err
		}
		stats = append(stats, stat)
	}

	// confirm before really collect

	if err := confirmStats(stats); err != nil {
		return err
	}

	// run collectors
	for _, c := range collectors {
		fmt.Printf("Collecting %s...\n", c.Desc())
		if err := c.Collect(&topo); err != nil {
			return err
		}
	}

	fmt.Printf("Collected data are stored in %s\n", color.CyanString(resultDir))
	return nil
}

func confirmStats(stats []map[string][]CollectStat) error {
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
