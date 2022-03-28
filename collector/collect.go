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
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	pingcapv1alpha1 "github.com/pingcap/diag/k8s/apis/pingcap/v1alpha1"
	kubetls "github.com/pingcap/diag/k8s/apis/tls"
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/logger"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/tui"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// types of data to collect
const (
	CollectTypeSystem        = "system"
	CollectTypeMonitor       = "monitor"
	CollectTypeLog           = "log"
	CollectTypeConfig        = "config"
	CollectTypeSchema        = "db_vars"
	CollectTypePerf          = "perf"
	CollectTypeAudit         = "audit_log"
	CollectTypeComponentMeta = "component_meta"

	CollectModeTiUP = "tiup-cluster"  // collect from a tiup-cluster deployed cluster
	CollectModeK8s  = "tidb-operator" // collect from a tidb-operator deployed cluster

)

var CollectDefaultSet = set.NewStringSet(
	CollectTypeSystem,
	CollectTypeMonitor,
	CollectTypeLog,
	CollectTypeConfig,
	CollectTypeAudit,
)

var CollectAdditionSet = set.NewStringSet(
	CollectTypeSchema,
	CollectTypePerf,
)

// Collector is the configuration defining an collecting job
type Collector interface {
	Prepare(*Manager, *models.TiDBCluster) (map[string][]CollectStat, error)
	Collect(*Manager, *models.TiDBCluster) error
	GetBaseOptions() *BaseOptions
	SetBaseOptions(*BaseOptions)
	Desc() string // a brief self description
}

// BaseOptions contains the options for check command
type BaseOptions struct {
	Cluster     string                  // cluster name
	Namespace   string                  // k8s namespace of the cluster
	User        string                  // username to login to the SSH server
	UsePassword bool                    // use password instead of identity file for ssh connection
	SSH         *tui.SSHConnectionProps // SSH credentials
	ScrapeBegin string                  // start timepoint when collecting metrics and logs
	ScrapeEnd   string                  // stop timepoint when collecting metrics and logs
}

// CollectOptions contains the options defining which type of data to collect
type CollectOptions struct {
	RawRequest    interface{}   // raw collect command or request
	Mode          string        // the cluster is deployed with what type of tool
	Include       set.StringSet // types of data to collect
	Exclude       set.StringSet // types of data not to collect
	MetricsFilter []string      // prefix of metrics to collect"
	Dir           string        // target directory to store collected data
	Limit         int           // rate limit of SCP
	PerfDuration  int           //seconds: profile time(s), default is 30s.
	CompressScp   bool          // compress of files during collecting
	ExitOnError   bool          // break the process and exit when an error occur
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
	kubeCli *kubernetes.Clientset,
	dynCli dynamic.Interface,
	skipConfirm bool,
) (string, error) {
	m.mode = cOpt.Mode

	var cls *models.TiDBCluster
	var tc *pingcapv1alpha1.TidbCluster
	var tm *pingcapv1alpha1.TidbMonitor
	var tlsCfg *tls.Config
	var err error
	switch cOpt.Mode {
	case CollectModeTiUP:
		cls, err = buildTopoForTiUPCluster(m, opt)
		if err != nil {
			return "", err
		}
		// get tls config
		tlsCfg, err = cls.Attributes[CollectModeTiUP].(spec.Topology).
			TLSConfig(m.specManager.Path(opt.Cluster, spec.TLSCertKeyDir))
		if err != nil {
			return "", err
		}
	case CollectModeK8s:
		cls, tc, tm, err = buildTopoForK8sCluster(m, opt, kubeCli, dynCli)
		if err != nil {
			return "", err
		}
		if tc.Spec.TLSCluster.Enabled {
			tlsCfg, err = kubetls.GetClusterClientTLSConfig(kubeCli, opt.Namespace, opt.Cluster, time.Duration(gOpt.APITimeout))
			klog.Infof("get tls config from secrets success")
		}
	default:
		return "", fmt.Errorf("unknown collect mode '%s'", cOpt.Mode)
	}
	if err != nil {
		return "", err
	}
	if cls == nil {
		return "", fmt.Errorf("no valid cluster topology parsed")
	}

	// prepare for different deploy mode
	var resultDir string
	var prompt string
	switch cOpt.Mode {
	case CollectModeTiUP:
		prompt, resultDir, err = m.prepareArgsForTiUPCluster(opt, cOpt)
	case CollectModeK8s:
		resultDir, err = m.prepareArgsForK8sCluster(opt, cOpt)
	}
	if err != nil {
		return "", err
	}

	collectorSet := map[string]bool{
		CollectTypeSystem:        false,
		CollectTypeMonitor:       false,
		CollectTypeLog:           false,
		CollectTypeConfig:        false,
		CollectTypeSchema:        false,
		CollectTypePerf:          false,
		CollectTypeAudit:         false,
		CollectTypeComponentMeta: false,
	}

	for name := range collectorSet {
		if canCollect(cOpt, name) {
			collectorSet[name] = true
		}
	}

	// build collector list
	collectors := make([]Collector, 0)

	// collect data from monitoring system
	collectors = append(collectors, &MetaCollectOptions{ // cluster metadata, always collected
		BaseOptions: opt,
		opt:         gOpt,
		rawRequest:  cOpt.RawRequest,
		session:     m.session,
		collectors:  collectorSet,
		resultDir:   resultDir,
		tc:          tc,
		tm:          tm,
		tlsCfg:      tlsCfg,
	})

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
				filter:      cOpt.MetricsFilter,
			},
		)
	}

	// populate SSH credentials if needed
	if m.mode == CollectModeTiUP && (canCollect(cOpt, CollectTypeSystem) ||
		canCollect(cOpt, CollectTypeLog) ||
		canCollect(cOpt, CollectTypeConfig)) {
		// collect data from remote servers
		var sshConnProps *tui.SSHConnectionProps = &tui.SSHConnectionProps{}
		if gOpt.SSHType != executor.SSHTypeNone {
			var err error
			if sshConnProps, err = tui.ReadIdentityFileOrPassword(opt.SSH.IdentityFile, opt.UsePassword); err != nil {
				return "", err
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
				compress:    cOpt.CompressScp,
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
				compress:    cOpt.CompressScp,
				tlsCfg:      tlsCfg,
			})
	}

	if canCollect(cOpt, CollectTypeSchema) {
		var user string
		fmt.Print("please enter database username:")
		fmt.Scanln(&user)
		password := tui.PromptForPassword("please enter database password:")
		collectors = append(collectors,
			&SchemaCollectOptions{
				BaseOptions: opt,
				opt:         gOpt,
				dbuser:      user,
				dbpasswd:    password,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
			})
	}

	if canCollect(cOpt, CollectTypePerf) {
		if cOpt.PerfDuration < 1 {
			if m.mode == CollectModeK8s {
				cOpt.PerfDuration = 30
			} else {
				return "", errors.Errorf("perf-duration cannot be less than 1")
			}

		}
		collectors = append(collectors,
			&PerfCollectOptions{
				BaseOptions: opt,
				opt:         gOpt,
				duration:    cOpt.PerfDuration,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
				tlsCfg:      tlsCfg,
			})
	}

	if canCollect(cOpt, CollectTypeAudit) {
		topoType := "cluster"
		if m.sysName == "dm" {
			topoType = m.sysName
		}
		collectors = append(collectors,
			&AuditLogCollectOptions{
				BaseOptions: opt,
				opt:         gOpt,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
				topoType:    topoType,
			})
	}

	if canCollect(cOpt, CollectTypeComponentMeta) {
		collectors = append(collectors,
			&ComponentMetaCollectOptions{
				BaseOptions: opt,
				opt:         gOpt,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
				tlsCfg:      tlsCfg,
				topo:        cls,
			})
	}

	// prepare
	// run collectors
	prepareErrs := make(map[string]error)
	stats := make([]map[string][]CollectStat, 0)
	for _, c := range collectors {
		m.logger.Infof("Detecting %s...\n", c.Desc())
		stat, err := c.Prepare(m, cls)
		if err != nil {
			if cOpt.ExitOnError {
				return "", err
			}
			msg := fmt.Sprintf("Error collecting %s: %s, the data might be incomplete.", c.Desc(), err)
			if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
				fmt.Println(color.YellowString(msg))
			} else {
				m.logger.Warnf(msg)
			}
			prepareErrs[c.Desc()] = err
		}
		stats = append(stats, stat)
	}

	// confirm before really collect
	if m.mode == CollectModeTiUP {
		fmt.Println(prompt)
		if err := confirmStats(stats, resultDir, skipConfirm); err != nil {
			return "", err
		}
	}

	err = os.MkdirAll(resultDir, 0755)
	if err != nil {
		return "", err
	}

	m.collectLock(resultDir)

	defer logger.OutputAuditLogToFileIfEnabled(resultDir, "diag_audit.log")

	// run collectors
	collectErrs := make(map[string]error)
	for _, c := range collectors {
		m.logger.Infof("Collecting %s...\n", c.Desc())
		if err := c.Collect(m, cls); err != nil {
			if cOpt.ExitOnError {
				return "", err
			}
			msg := fmt.Sprintf("Error collecting %s: %s, the data might be incomplete.", c.Desc(), err)
			if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
				fmt.Println(color.YellowString(msg))
			} else {
				m.logger.Warnf(msg)
			}
			collectErrs[c.Desc()] = err
		}
	}

	if len(collectErrs) > 0 {
		if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
			fmt.Println(color.RedString("Some errors occurred during the process, please check if data needed are complete:"))
		}
		for k, v := range prepareErrs {
			m.logger.Errorf("%s:\t%s\n", k, v)
		}
		for k, v := range collectErrs {
			m.logger.Errorf("%s:\t%s\n", k, v)
		}
	}

	m.collectUnlock(resultDir)

	dir := resultDir
	if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
		dir = color.CyanString(resultDir)
	}
	m.logger.Infof("Collected data are stored in %s\n", dir)
	return resultDir, nil
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

func confirmStats(stats []map[string][]CollectStat, resultDir string, skipConfirm bool) error {
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

	if skipConfirm {
		return nil
	}
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
