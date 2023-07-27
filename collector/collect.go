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
	"reflect"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pingcap/diag/pkg/models"
	kubetls "github.com/pingcap/diag/pkg/tls"
	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/errors"
	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tiup/pkg/cluster/executor"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/logger"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
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
	CollectTypeDebug         = "debug"
	CollectTypeComponentMeta = "component_meta"
	CollectTypeBind          = "sql_bind"
	CollectTypePlanReplayer  = "plan_replayer"

	CollectModeTiUP   = "tiup-cluster"  // collect from a tiup-cluster deployed cluster
	CollectModeK8s    = "tidb-operator" // collect from a tidb-operator deployed cluster
	CollectModeManual = "manual"        // collect from a manually deployed cluster
	DiagModeCmd       = "cmd"           // run diag collect at command line mode
	DiagModeServer    = "server"        // run diag collect at server mode

	AttrKeyPromEndpoint = "prometheus-endpoint"
	AttrKeyClusterID    = "cluster-id"
	AttrKeyPDEndpoint   = "pd-endpoint"
	AttrKeyTiDBHost     = "tidb-host"
	AttrKeyTiDBPort     = "tidb-port"
	AttrKeyTiDBStatus   = "tidb-status-port"
	AttrKeyTLSCAFile    = "tls-ca-file"
	AttrKeyTLSCertFile  = "tls-cert-file"
	AttrKeyTLSKeyFile   = "tls-privkey-file"
)

type CollectTree struct {
	System        bool
	Monitor       collectMonitor
	Log           collectLog
	Config        collectConfig
	DBVars        bool
	Perf          bool
	Debug         bool
	ComponentMeta bool
	SQLBind       bool
	PlanReplayer  bool
}

// Collector is the configuration defining an collecting job
type Collector interface {
	Prepare(*Manager, *models.TiDBCluster) (map[string][]CollectStat, error)
	Collect(*Manager, *models.TiDBCluster) error
	GetBaseOptions() *BaseOptions
	SetBaseOptions(*BaseOptions)
	Desc() string // a brief self description
	Close()
}

// BaseOptions contains the options for check command
type BaseOptions struct {
	Cluster          string                  // cluster name
	Namespace        string                  // k8s namespace of the cluster
	MonitorNamespace string                  // k8s namespace of the monitor
	Kubeconfig       string                  // path of kubeconfig
	User             string                  // username to login to the SSH server
	UsePassword      bool                    // use password instead of identity file for ssh connection
	SSH              *tui.SSHConnectionProps // SSH credentials
	ScrapeBegin      string                  // start timepoint when collecting metrics and logs
	ScrapeEnd        string                  // stop timepoint when collecting metrics and logs
}

// CollectOptions contains the options defining which type of data to collect
type CollectOptions struct {
	RawRequest      interface{}       // raw collect command or request
	Mode            string            // the cluster is deployed with what type of tool
	DiagMode        string            // run diag collect at command line mode or server mode
	ProfileName     string            // the name of a pre-defined collecting profile
	Collectors      CollectTree       // struct to show which collector is enabled
	MetricsFilter   []string          // prefix of metrics to collect"
	MetricsLabel    map[string]string // label to filte metrics
	Dir             string            // target directory to store collected data
	Limit           int               // rate limit of SCP
	MetricsLimit    int               // query limit of one request
	PerfDuration    int               //seconds: profile time(s), default is 30s.
	CompressScp     bool              // compress of files during collecting
	CompressMetrics bool              // compress of files during collecting
	RawMonitor      bool              // collect raw data for metrics
	ExitOnError     bool              // break the process and exit when an error occur
	ExtendedAttrs   map[string]string // extended attributes used for manual collecting mode
	ExplainSQLPath  string            // File path for explain sql
	ExplainSqls     []string          // explain sqls
	CurrDB          string
	Header          []string
	UsePortForward  bool // use portforward when call api inside k8s cluster
}

// CollectStat is estimated size stats of data to be collected
type CollectStat struct {
	Target     string
	Size       int64
	Attributes map[string]interface{}
}

func (c BaseOptions) Close() {
	return
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
	m.diagMode = cOpt.DiagMode
	m.mode = cOpt.Mode

	var sensitiveTag bool
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
		if tc != nil && tc.Spec.TLSCluster.Enabled {
			tlsCfg, err = kubetls.GetClusterClientTLSConfig(kubeCli, opt.Namespace, opt.Cluster, time.Second*time.Duration(gOpt.APITimeout))
			if err != nil {
				return "", err
			}
			klog.Infof("get tls config from secrets success")
		}
	case CollectModeManual:
		cls, err = buildTopoForManualCluster(cOpt)
		if err != nil {
			return "", err
		}
		tlsCfg, err = tlsConfig(
			cOpt.ExtendedAttrs[AttrKeyTLSCAFile],
			cOpt.ExtendedAttrs[AttrKeyTLSCertFile],
			cOpt.ExtendedAttrs[AttrKeyTLSKeyFile],
		)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unknown collect mode '%s'", cOpt.Mode)
	}
	if cls == nil {
		return "", fmt.Errorf("no valid cluster topology parsed")
	}

	// prepare for different deploy mode
	var resultDir string
	var prompt string
	switch cOpt.Mode {
	case CollectModeTiUP,
		CollectModeManual:
		prompt, resultDir, err = m.prepareArgsForTiUPCluster(opt, cOpt)
	case CollectModeK8s:
		resultDir, err = m.prepareArgsForK8sCluster(opt, cOpt)
	}
	if err != nil {
		return "", err
	}

	if cOpt.ProfileName != "" {
		cp, err := readProfile(cOpt.ProfileName)
		if err != nil {
			return "", errors.Annotate(err, "failed to load profile from file")
		}
		msg := fmt.Sprintf(
			"Apply configs from profile %s(%s) %s",
			cp.Name, cOpt.ProfileName, cp.Version,
		)
		if len(cp.Maintainers) > 0 {
			msg = fmt.Sprintf("%s by %s", msg, strings.Join(cp.Maintainers, ","))
		}
		logprinter.Infof(msg)

		cOpt.Collectors, err = ParseCollectTree(cp.Collectors, nil)
		if err != nil {
			return "", err
		}
		gOpt.Roles = append(gOpt.Roles, cp.Roles...)
		cOpt.MetricsFilter = append(cOpt.MetricsFilter, cp.MetricFilters...)
	}

	var explainSqls []string
	if len(cOpt.ExplainSqls) > 0 {
		explainSqls = cOpt.ExplainSqls
	} else if len(cOpt.ExplainSQLPath) > 0 {
		b, err := os.ReadFile(cOpt.ExplainSQLPath)
		if err != nil {
			return "", err
		}
		sqls := strings.Split(string(b), ";")
		for _, sql := range sqls {
			if len(sql) > 0 {
				explainSqls = append(explainSqls, sql)
			}
		}
		cOpt.Collectors.PlanReplayer = true
	} else {
		if cOpt.Collectors.PlanReplayer {
			return "", errors.New("explain-sql should be set if PlanReplayer is included")
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
		collectors:  cOpt.Collectors,
		resultDir:   resultDir,
		tc:          tc,
		tm:          tm,
		tlsCfg:      tlsCfg,
	})

	// collect data from monitoring system
	if canCollect(&cOpt.Collectors.Monitor.Alert) {
		collectors = append(collectors,
			&AlertCollectOptions{ // alerts
				BaseOptions: opt,
				opt:         gOpt,
				resultDir:   resultDir,
				compress:    cOpt.CompressMetrics,
			})
	}
	if canCollect(&cOpt.Collectors.Monitor.Metric) && !cOpt.RawMonitor {
		collectors = append(collectors,
			&MetricCollectOptions{ // metrics
				BaseOptions:  opt,
				opt:          gOpt,
				resultDir:    resultDir,
				label:        cOpt.MetricsLabel,
				filter:       cOpt.MetricsFilter,
				limit:        cOpt.MetricsLimit,
				compress:     cOpt.CompressMetrics,
				customHeader: cOpt.Header,
				portForward:  cOpt.UsePortForward,
			},
		)
	}
	if canCollect(&cOpt.Collectors.Monitor.Metric) && cOpt.RawMonitor {
		collectors = append(collectors,
			&TSDBCollectOptions{ // metrics
				BaseOptions: opt,
				opt:         gOpt,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
				limit:       cOpt.Limit,
				compress:    cOpt.CompressScp,
			},
		)
	}

	// populate SSH credentials if needed
	if (m.mode == CollectModeTiUP || m.mode == CollectModeManual) &&
		(canCollect(&cOpt.Collectors.System) ||
			canCollect(&cOpt.Collectors.Log) ||
			canCollect(&cOpt.Collectors.Config)) {
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

	if canCollect(&cOpt.Collectors.System) {
		collectors = append(collectors, &SystemCollectOptions{
			BaseOptions: opt,
			opt:         gOpt,
			resultDir:   resultDir,
		})
	}

	// collect log files
	if canCollect(&cOpt.Collectors.Log) {
		collectors = append(collectors,
			&LogCollectOptions{
				BaseOptions: opt,
				opt:         gOpt,
				collector:   cOpt.Collectors.Log,
				limit:       cOpt.Limit,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
				compress:    cOpt.CompressScp,
				kubeCli:     kubeCli,
			})
	}

	// collect config files
	if canCollect(&cOpt.Collectors.Config) {
		collectors = append(collectors,
			&ConfigCollectOptions{
				BaseOptions: opt,
				Collectors:  cOpt.Collectors.Config,
				opt:         gOpt,
				limit:       cOpt.Limit,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
				compress:    cOpt.CompressScp,
				tlsCfg:      tlsCfg,
			})
	}

	var dbUser, dbPassword string
	if needDBKey(cOpt.Collectors) {
		fmt.Print("please enter database username:")
		fmt.Scanln(&dbUser)
		dbPassword = tui.PromptForPassword("please enter database password:")
	}

	if canCollect(&cOpt.Collectors.DBVars) {
		collectors = append(collectors,
			&SchemaCollectOptions{
				BaseOptions: opt,
				opt:         gOpt,
				dbuser:      dbUser,
				dbpasswd:    dbPassword,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
			})
	}

	if canCollect(&cOpt.Collectors.SQLBind) {
		collectors = append(collectors,
			&BindCollectOptions{
				BaseOptions: opt,
				opt:         gOpt,
				dbuser:      dbUser,
				dbpasswd:    dbPassword,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
			})
	}

	// hide perf
	if canCollect(&cOpt.Collectors.Perf) {
		return "", fmt.Errorf("perf collection is disabled in diag, use TiDB Dashboard instead")
	}

	// 	if len(cls.TiKV) > 0 {
	// 		// maybe it's better to use tiup/pkg/tidbver
	// 		if !(semver.Compare(cls.Version, "v5.0.0") >= 0 || strings.Contains(cls.Version, "nightly")) {
	// 			return "", errors.Errorf("cannot collect perf information of Tikv whose version is less than v5.0.0")
	// 		}
	// 	}

	// 	if cOpt.PerfDuration < 1 {
	// 		if m.mode == CollectModeK8s {
	// 			cOpt.PerfDuration = 30
	// 		} else {
	// 			return "", errors.Errorf("perf-duration cannot be less than 1")
	// 		}

	// 	}
	// 	collectors = append(collectors,
	// 		&PerfCollectOptions{
	// 			BaseOptions: opt,
	// 			opt:         gOpt,
	// 			duration:    cOpt.PerfDuration,
	// 			resultDir:   resultDir,
	// 			fileStats:   make(map[string][]CollectStat),
	// 			tlsCfg:      tlsCfg,
	// 		})
	// }

	// todo: rename dir name to ops and move functions to log.go
	if canCollect(&cOpt.Collectors.Log.Ops) {
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

	if canCollect(&cOpt.Collectors.ComponentMeta) {
		sensitiveTag = true
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

	if canCollect(&cOpt.Collectors.Debug) {
		collectors = append(collectors,
			&DebugCollectOptions{
				BaseOptions: opt,
				opt:         gOpt,
				resultDir:   resultDir,
				fileStats:   make(map[string][]CollectStat),
				tlsCfg:      tlsCfg,
			})
	}

	if canCollect(&cOpt.Collectors.PlanReplayer) {
		collectors = append(collectors,
			&PlanReplayerCollectorOptions{
				BaseOptions:    opt,
				opt:            gOpt,
				dbuser:         dbUser,
				dbpasswd:       dbPassword,
				resultDir:      resultDir,
				sqls:           explainSqls,
				tlsCfg:         tlsCfg,
				tables:         make(map[table]struct{}),
				views:          make(map[table]struct{}),
				tablesAndViews: make(map[table]struct{}),
				currDB:         cOpt.CurrDB,
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
		defer c.Close()
		stats = append(stats, stat)
	}

	// confirm before really collect
	switch m.diagMode {
	case DiagModeCmd:
		fmt.Println(prompt)
		if err := confirmStats(stats, resultDir, sensitiveTag, skipConfirm); err != nil {
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
func (m *Manager) getOutputDir(dir, clusterName string) (string, error) {
	if dir == "" {
		dir = filepath.Join(".", fmt.Sprintf("diag-%s-%s", clusterName, m.session))
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

func confirmStats(stats []map[string][]CollectStat, resultDir string, sensitiveTag, skipConfirm bool) error {
	fmt.Printf("Estimated size of data to collect:\n")
	var total, compressed int64
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
				if strings.HasSuffix(s.Target, "metrics, compressed") {
					// metrics are already compressed
					compressed += s.Size
				} else {
					compressed += s.Size / 10
				}
				statTable = append(statTable, []string{host, color.CyanString(readableSize(s.Size)), s.Target})
			}
		}
	}
	statTable = append(statTable, []string{"Total", color.YellowString(readableSize(total)), "(inaccurate)"})
	tui.PrintTable(statTable, true)

	if sensitiveTag {
		fmt.Println(color.HiRedString("This collect action may contain sensitive data, please do not use it in production environment"))
	}

	fmt.Printf("These data will be stored in %s\n", color.CyanString(resultDir))

	if compressed > 3*1024*1024*1024 {
		fmt.Println(color.YellowString("The amount of data collected is large, and the compressed data may be larger than 3GB, which exceeds the upload file size limit. Suggest to shorten the collection period if need to upload collected data to Clinic server."))
	}

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

/*
func canCollect(cOpt *CollectOptions, cType string) bool {
	return cOpt.Include.Exist(cType) && !cOpt.Exclude.Exist(cType)
}
*/

func canCollect(w interface{}) bool {
	reflectV := reflect.ValueOf(w).Elem()
	return utils.RecursiveIfBoolValueExist(reflectV, true)
}

func needDBKey(c CollectTree) bool {
	return c.SQLBind || c.DBVars || c.PlanReplayer
}

func ParseCollectTree(include, exclude []string) (CollectTree, error) {
	var collectWhat CollectTree
	for _, item := range include {
		reflectV := reflect.ValueOf(&collectWhat).Elem()
		keys := strings.Split(item, ".")
		for _, k := range keys {
			if reflectV.Kind() != reflect.Struct {
				return collectWhat, fmt.Errorf("%s is not a valid diag collection type", item)
			}
			num := reflectV.NumField()
			var i int
			for i = 0; i < num; i++ {
				if strings.ToLower(k) == strings.ToLower(reflectV.Type().Field(i).Name) {
					reflectV = reflectV.Field(i)
					break
				}
			}
			if i == num {
				return collectWhat, fmt.Errorf("%s is not a valid diag collection type", item)
			}
		}
		utils.RecursiveSetBoolValue(reflectV, true)
	}

	for _, item := range exclude {
		reflectV := reflect.ValueOf(&collectWhat).Elem()
		keys := strings.Split(item, ".")
		for _, k := range keys {
			if reflectV.Kind() != reflect.Struct {
				return collectWhat, fmt.Errorf("%s is not a valid diag collection type", item)
			}
			num := reflectV.NumField()
			var i int
			for i = 0; i < num; i++ {
				if strings.ToLower(k) == strings.ToLower(reflectV.Type().Field(i).Name) {
					reflectV = reflectV.Field(i)
					break
				}
			}
			if i == num {
				return collectWhat, fmt.Errorf("%s is not a valid diag collection type", item)
			}
		}
		utils.RecursiveSetBoolValue(reflectV, false)
	}
	return collectWhat, nil
}

func (t CollectTree) List() []string {
	var r func(reflect.Value, string) []string
	r = func(reflectV reflect.Value, path string) (result []string) {
		switch reflectV.Kind() {
		case reflect.Struct:
			for i := 0; i < reflectV.NumField(); i++ {
				childpath := strings.ToLower(reflectV.Type().Field(i).Name)
				if path != "" {
					childpath = fmt.Sprintf("%s.%s", path, childpath)
				}
				result = append(result, r(reflectV.Field(i), childpath)...)
			}
		case reflect.Bool:
			if reflectV.Bool() {
				result = []string{path}
			}
		}
		return result
	}
	return r(reflect.ValueOf(&t).Elem(), "")
}
