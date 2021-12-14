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

package sourcedata

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/checker/config"
	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/diag/collector"
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/errors"
	"github.com/pingcap/log"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

type Fetcher interface {
	FetchData(rules *config.RuleSpec) (*proto.SourceDataV2, proto.RuleSet, error)
}

const (
	ConfigFlag CheckFlag = 1 << iota // rules summarized from on-call issues.
	PerformanceFlag
	DefaultConfigFlag // rules check default value.
)

type CheckFlag int

func (cf CheckFlag) checkConfig() bool {
	return cf&ConfigFlag > 0
}

func (cf CheckFlag) checkPerformance() bool {
	return cf&PerformanceFlag > 0
}

func (cf CheckFlag) checkDefaultConfig() bool {
	return cf&DefaultConfigFlag > 0
}

// FileFetcher load all needed data from file
type FileFetcher struct {
	dataDirPath string // dataDirPath point to a folder
	checkFlag   CheckFlag
	outDirPath  string
	// set during processing
	clusterMeta *spec.ClusterMeta
	clusterJSON *collector.ClusterJSON
}

type FileFetcherOpt func(ff *FileFetcher) error

func WithCheckFlag(flags ...CheckFlag) FileFetcherOpt {
	return func(ff *FileFetcher) error {
		for _, flag := range flags {
			ff.checkFlag |= flag
		}
		return nil
	}
}

func WithOutputDir(outDir string) FileFetcherOpt {
	return func(ff *FileFetcher) error {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return err
		}
		ff.outDirPath = outDir
		return nil
	}
}

func NewFileFetcher(dataDirPath string, opts ...FileFetcherOpt) (*FileFetcher, error) {
	ff := &FileFetcher{
		dataDirPath: dataDirPath,
		checkFlag:   0,
	}
	for _, opt := range opts {
		if err := opt(ff); err != nil {
			return nil, err
		}
	}
	return ff, nil
}

// FetchData retrieve config data from file path, and filter rules by component version
// dataPath is the path to the top folder which contain the data collected by diag collect.
func (f *FileFetcher) FetchData(rules *config.RuleSpec) (*proto.SourceDataV2, proto.RuleSet, error) {
	if err := f.loadClusterMetaData(); err != nil {
		return nil, nil, err
	}
	clusterVersion, err := f.getClusterVersion()
	if err != nil {
		return nil, nil, err
	}
	filterFunc := func(item config.RuleItem) (bool, error) {
		// filter on version
		ok, err := item.Version.Contain(clusterVersion)
		if err != nil {
			return false, err
		} else if !ok {
			return false, nil
		}
		// filter on check type
		switch item.CheckType {
		case proto.DefaultConfigType:
			return f.checkFlag.checkDefaultConfig(), nil
		case proto.PerformanceType:
			return f.checkFlag.checkPerformance(), nil
		case proto.ConfigType:
			return f.checkFlag.checkConfig(), nil
		}
		return false, nil
	}
	// rSet contain all rules match the cluster version
	rSet, err := rules.FilterOn(filterFunc)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, err
	}
	sourceData := &proto.SourceDataV2{
		ClusterInfo:   f.clusterJSON,
		NodesData:     make(map[string][]proto.Config),
		TidbVersion:   clusterVersion,
		DashboardData: &proto.DashboardData{},
	}
	ctx := context.Background()
	// decode config.json for related config component
	if f.checkFlag.checkConfig() || f.checkFlag.checkDefaultConfig() {
		if err := f.loadRealTimeConfig(ctx, sourceData, rSet); err != nil {
			return nil, nil, err
		}
	}
	// decode sql performance data
	if f.checkFlag.checkPerformance() {
		// TODO: check if there is any performance rule before load slow log
		{
			reader, err := os.Open(f.genInfoSchemaCSVPath("mysql.tidb.csv"))
			if err != nil {
				return nil, nil, err
			}
			defer reader.Close()
			data, err := f.loadSysVariables(reader)
			if err != nil {
				return nil, nil, err
			}
			gcLifeTime, err := time.ParseDuration(data["tikv_gc_life_time"])
			if err != nil {
				return nil, nil, err
			}
			sourceData.DashboardData.OldVersionProcesskey.GcLifeTime = int(gcLifeTime.Nanoseconds() / 1e9)
		}
		if err := f.loadSlowLog(ctx, sourceData); err != nil {
			return nil, nil, err
		}
	}

	return sourceData, rSet, nil
}

// sourceData will be updated
func (f *FileFetcher) loadRealTimeConfig(_ context.Context, sourceData *proto.SourceDataV2, rSet proto.RuleSet) error {
	nameStructs := rSet.GetNameStructs()
	for name := range nameStructs {
		switch name {
		case proto.PdComponentName:
			for _, spec := range f.getComponents(proto.PdComponentName) {
				// todo add no data
				host := spec.Host()
				if pod, ok := spec.Attributes()["pod"].(string); ok {
					host = pod
				}
				cfgPath := path.Join(f.dataDirPath, host, "conf", "config.json")
				if deployDir, ok := spec.Attributes()["deploy_dir"].(string); ok {
					cfgPath = path.Join(f.dataDirPath, host, deployDir, "conf", "config.json")
				}
				cfg := proto.NewPdConfigData()
				bs, err := os.ReadFile(cfgPath)
				if err != nil {
					cfg.PdConfig = nil // skip error
				} else {
					if err := json.Unmarshal(bs, cfg); err != nil {
						logrus.Error(err)
						return err
					}
				}
				cfg.Host = spec.Host()
				cfg.Port = spec.MainPort()
				sourceData.AppendConfig(cfg, proto.PdComponentName)
			}
		case proto.TikvComponentName:
			for _, spec := range f.getComponents(proto.TikvComponentName) {
				host := spec.Host()
				if pod, ok := spec.Attributes()["pod"].(string); ok {
					host = pod
				}
				cfgPath := path.Join(f.dataDirPath, host, "conf", "config.json")
				if deployDir, ok := spec.Attributes()["deploy_dir"].(string); ok {
					cfgPath = path.Join(f.dataDirPath, host, deployDir, "conf", "config.json")
				}
				cfg := proto.NewTikvConfigData()
				bs, err := os.ReadFile(cfgPath)
				if err != nil {
					cfg.TikvConfig = nil // skip error
				} else {
					if err := json.Unmarshal(bs, cfg); err != nil {
						logrus.Error(err)
						return err
					}
					cfg.LoadClusterMeta(f.clusterJSON.Topology, f.clusterMeta)
				}
				cfg.Host = spec.Host()
				cfg.Port = spec.MainPort()
				sourceData.AppendConfig(cfg, proto.TikvComponentName)
			}
		case proto.TidbComponentName:
			for _, spec := range f.getComponents(proto.TidbComponentName) {
				host := spec.Host()
				if pod, ok := spec.Attributes()["pod"].(string); ok {
					host = pod
				}
				cfgPath := path.Join(f.dataDirPath, host, "conf", "config.json")
				if deployDir, ok := spec.Attributes()["deploy_dir"].(string); ok {
					cfgPath = path.Join(f.dataDirPath, host, deployDir, "conf", "config.json")
				}
				cfg := proto.NewTidbConfigData()
				bs, err := os.ReadFile(cfgPath)
				if err != nil {
					cfg.TidbConfig = nil
				} else {
					if err := json.Unmarshal(bs, cfg); err != nil {
						logrus.Error(err)
						return err
					}
				}
				cfg.Host = spec.Host()
				cfg.Port = spec.MainPort()
				sourceData.AppendConfig(cfg, proto.TidbComponentName)
			}
		default:
		}
	}
	return nil
}

// todo sourceData will be updated
func (f *FileFetcher) loadSlowLog(ctx context.Context, sourceData *proto.SourceDataV2) (err error) {
	header := []string{"Time", "Digest", "Plan_digest", "Process_time", "Process_keys", "Rocksdb_delete_skipped_count", "Total_keys"}
	idxLookUp := NewIdxLookup(header)
	avgProcessTimePlanAcc, err := NewAvgProcessTimePlanAccumulator(idxLookUp)
	if err != nil {
		return err
	} else if len(f.outDirPath) > 0 {
		csvFile, err := os.Create(path.Join(f.outDirPath, "poor_effective_plan.csv"))
		if err != nil {
			return err
		}
		defer csvFile.Close()
		avgProcessTimePlanAcc.setCSVWriter(csvFile)
	}

	scanOldVersionPlanAcc, err := NewScanOldVersionPlanAccumulator(idxLookUp)
	if err != nil {
		return err
	} else if len(f.outDirPath) > 0 {
		csvFile, err := os.Create(path.Join(f.outDirPath, "old_version_count.csv"))
		if err != nil {
			return err
		}
		defer csvFile.Close()
		scanOldVersionPlanAcc.setCSVWriter(csvFile)
	}

	skipDeletedCntPlanAcc, err := NewSkipDeletedCntPlanAccumulator(idxLookUp)
	if err != nil {
		return err
	} else if len(f.outDirPath) > 0 {
		csvFile, err := os.Create(path.Join(f.outDirPath, "scan_key_skip.csv"))
		if err != nil {
			return err
		}
		defer csvFile.Close()
		skipDeletedCntPlanAcc.setCSVWriter(csvFile)
	}
	closers := make([]io.Closer, 0)
	var cnt int64
	for _, spec := range f.getComponents(proto.TidbComponentName) {
		slowLogPath := path.Join(f.dataDirPath, spec.Host(), "log", "tidb_slow_query.log")
		if deployDir, ok := spec.Attributes()["deploy_dir"]; ok {
			slowLogPath = path.Join(f.dataDirPath, spec.Host(), deployDir.(string), "log", "tidb_slow_query.log")
		}
		// todo
		retriever, err := NewSlowQueryRetriever(5, time.Local, header, slowLogPath, WithTimeRanges(time.Now().AddDate(0, 0, -7), time.Now()))
		if err != nil {
			return err
		}
		// todo, how to close
		closers = append(closers, retriever)
		for true {
			rows, err := retriever.retrieve(ctx)
			if err != nil {
				log.Warn("retrieve slow log failed", zap.Error(err))
				continue
			}
			// if return rows are empty, this file reaches end.
			if len(rows) == 0 {
				break
			}
			cnt += int64(len(rows))
			for _, row := range rows {
				if err := avgProcessTimePlanAcc.feed(row); err != nil {
					log.Warn("feed row to accumulator failed", zap.Error(err))
				}
				if err := scanOldVersionPlanAcc.feed(row); err != nil {
					log.Warn("feed row to accumulator failed", zap.Error(err))
				}
				if err := skipDeletedCntPlanAcc.feed(row); err != nil {
					log.Warn("feed row to accumulator failed", zap.Error(err))
				}
			}
		}
	}
	for _, c := range closers {
		c.Close()
	}
	avgProcessTimePlanAcc.debugState()
	scanOldVersionPlanAcc.debugState()
	skipDeletedCntPlanAcc.debugState()
	sourceData.DashboardData.ExecutionPlanInfoList, err = avgProcessTimePlanAcc.build()
	if err != nil {
		return err
	}
	{
		digestPair, err := scanOldVersionPlanAcc.build()
		if err != nil {
			return err
		}
		sourceData.DashboardData.OldVersionProcesskey.Count = len(digestPair)
	}
	{
		digestPair, err := skipDeletedCntPlanAcc.build()
		if err != nil {
			return err
		}
		sourceData.DashboardData.TombStoneStatistics.Count = len(digestPair)
	}
	log.Debug("read all slow log", zap.Int64("row cnt", cnt))
	return nil
}

func (f *FileFetcher) loadSlowPlanData(reader io.Reader) (data map[string][2]proto.ExecutionPlanInfo, err error) {
	csvReader := csv.NewReader(reader)
	header, err := csvReader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	if len(header) == 0 {
		return nil, errors.New("invalid csv content")
	}
	idxLookUp := make(map[string]int)
	// todo
	for idx, col := range header {
		idxLookUp[strings.ToLower(col)] = idx
	}
	execPlan := make(map[string]*[2]proto.ExecutionPlanInfo)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		digest := record[idxLookUp["digest"]]
		planDigest := record[idxLookUp["plan_digest"]]
		avgPTime, err := strconv.ParseFloat(record[idxLookUp["avg_process_time"]], 64)
		if err != nil {
			return nil, err
		}
		lastTime, err := time.Parse("2006-01-02 15:04:05.999999", record[idxLookUp["last_time"]])
		if err != nil {
			return nil, err
		}
		avgPTimeSeconds := int64(avgPTime)
		lastTimeUnix := lastTime.Unix()
		if _, ok := execPlan[digest]; !ok {
			execPlan[digest] = &[2]proto.ExecutionPlanInfo{
				{PlanDigest: planDigest, AvgProcessTime: avgPTimeSeconds, MaxLastTime: lastTimeUnix},
				{PlanDigest: planDigest, AvgProcessTime: avgPTimeSeconds, MaxLastTime: lastTimeUnix},
			}
		} else {
			// check min and max
			if avgPTimeSeconds < execPlan[digest][0].AvgProcessTime {
				execPlan[digest][0].PlanDigest = planDigest
				execPlan[digest][0].AvgProcessTime = avgPTimeSeconds
				execPlan[digest][0].MaxLastTime = lastTimeUnix
			}
			if avgPTimeSeconds > execPlan[digest][1].AvgProcessTime {
				execPlan[digest][1].PlanDigest = planDigest
				execPlan[digest][1].AvgProcessTime = avgPTimeSeconds
				execPlan[digest][1].MaxLastTime = lastTimeUnix
			}
		}
	}
	data = make(map[string][2]proto.ExecutionPlanInfo)
	for digest, plan := range execPlan {
		data[digest] = *plan
	}
	return data, nil
}

func (f *FileFetcher) loadDigest(reader io.Reader) ([]proto.DigestPair, error) {
	csvReader := csv.NewReader(reader)
	header, err := csvReader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	if len(header) == 0 {
		return nil, errors.New("invalid csv content")
	}
	idxLookUp := make(map[string]int)
	for idx, col := range header {
		idxLookUp[strings.ToLower(col)] = idx
	}
	data := make([]proto.DigestPair, 0)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		digest := record[idxLookUp["digest"]]
		planDigest := record[idxLookUp["plan_digest"]]
		data = append(data, proto.DigestPair{
			Digest:     digest,
			PlanDigest: planDigest,
		})
	}
	return data, nil
}

func (f *FileFetcher) loadSysVariables(reader io.Reader) (map[string]string, error) {
	csvReader := csv.NewReader(reader)
	header, err := csvReader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	if len(header) != 2 {
		return nil, errors.New("invalid csv content")
	}
	data := make(map[string]string)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		variableName := record[0]
		variableValue := record[1]
		data[variableName] = variableValue
	}
	return data, nil
}

func (f *FileFetcher) genMetaConfigPath() string {
	return path.Join(f.dataDirPath, collector.FileNameTiUPClusterMeta)
}

func (f *FileFetcher) genClusterJSONPath() string {
	return path.Join(f.dataDirPath, collector.FileNameClusterJSON)
}

func (f *FileFetcher) genInfoSchemaCSVPath(fileName string) string {
	return path.Join(f.dataDirPath, collector.DirNameSchema, fileName)
}

// loadClusterMetaData must be called before getClusterVersion and getComponents
func (f *FileFetcher) loadClusterMetaData() error {
	clusterJSON := &collector.ClusterJSON{}

	bs, err := os.ReadFile(f.genClusterJSONPath())
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bs, clusterJSON); err != nil {
		return err
	}
	f.clusterJSON = clusterJSON
	if clusterJSON.Topology == nil {
		clusterMeta := &spec.ClusterMeta{}
		bs, err := os.ReadFile(f.genMetaConfigPath())
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(bs, clusterMeta); err != nil {
			return err
		}
		f.clusterMeta = clusterMeta
	}
	return nil
}

// use tidb image version for tidb on k8s, version in Topology may be `latest`
func (f *FileFetcher) getClusterVersion() (string, error) {
	if f.clusterJSON == nil {
		return "", errors.New("cluster json is nil")
	}
	if f.clusterJSON.Topology == nil {
		if f.clusterMeta == nil {
			return "", errors.New("cluster meta is nil")
		}
		return f.clusterMeta.Version, nil
	}
	// use Topology.Version first
	if _, err := semver.NewVersion(f.clusterJSON.Topology.Version); err == nil {
		return f.clusterJSON.Topology.Version, nil
	}
	// use tidb image version
	if len(f.clusterJSON.Topology.TiDB) == 0 {
		return "", errors.New("can not infer tidb version")
	}
	imageURL, ok := f.clusterJSON.Topology.TiDB[0].Attributes()["image"].(string)
	if !ok {
		return "", errors.New("can not infer tidb version")
	}
	imageTag := strings.Split(imageURL, ":")
	if len(imageTag) != 2 {
		return "", errors.New("can not infer tidb version")
	}
	mainVersion := strings.Split(imageTag[1], "-")[0]
	if !strings.HasPrefix(mainVersion, "v") {
		mainVersion = fmt.Sprintf("v%s", mainVersion)
	}
	return mainVersion, nil
}

func (f *FileFetcher) getComponents(name proto.ComponentName) []models.Component {
	components := make([]models.Component, 0)
	if f.clusterJSON != nil && f.clusterJSON.Topology != nil {
		switch name {
		case proto.PdComponentName:
			for _, pdSpec := range f.clusterJSON.Topology.PD {
				components = append(components, pdSpec)
			}
		case proto.TikvComponentName:
			for _, tikvSpec := range f.clusterJSON.Topology.TiKV {
				components = append(components, tikvSpec)
			}
		case proto.TidbComponentName:
			for _, tidbSpec := range f.clusterJSON.Topology.TiDB {
				components = append(components, tidbSpec)
			}
		}
		return components
	}
	if f.clusterMeta != nil && f.clusterMeta.Topology != nil {
		switch name {
		case proto.PdComponentName:
			for _, pdSpec := range f.clusterMeta.Topology.PDServers {
				compSpec := models.ComponentSpec{
					Host:       pdSpec.Host,
					Port:       pdSpec.ClientPort,
					StatusPort: pdSpec.PeerPort,
					SSHPort:    pdSpec.SSHPort,
					Attributes: map[string]interface{}{
						"deploy_dir": pdSpec.DeployDir,
					}}
				components = append(components, &models.PDSpec{ComponentSpec: compSpec})
			}
		case proto.TikvComponentName:
			for _, tikvSpec := range f.clusterMeta.Topology.TiKVServers {
				compSpec := models.ComponentSpec{
					Host:       tikvSpec.Host,
					Port:       tikvSpec.Port,
					StatusPort: tikvSpec.StatusPort,
					SSHPort:    tikvSpec.SSHPort,
					Attributes: map[string]interface{}{
						"deploy_dir": tikvSpec.DeployDir,
					}}
				components = append(components, &models.PDSpec{ComponentSpec: compSpec})
			}
		case proto.TidbComponentName:
			for _, tidbSpec := range f.clusterMeta.Topology.TiDBServers {
				compSpec := models.ComponentSpec{
					Host:       tidbSpec.Host,
					Port:       tidbSpec.Port,
					StatusPort: tidbSpec.StatusPort,
					SSHPort:    tidbSpec.SSHPort,
					Attributes: map[string]interface{}{
						"deploy_dir": tidbSpec.DeployDir,
					}}
				components = append(components, &models.PDSpec{ComponentSpec: compSpec})
			}
		}
		return components
	}
	return nil
}

func (f *FileFetcher) checkAvailable(name proto.ComponentName) bool {
	if f.clusterJSON != nil && f.clusterJSON.Topology != nil {
		switch name {
		case proto.PdComponentName:
			return len(f.clusterJSON.Topology.PD) > 0
		case proto.TikvComponentName:
			return len(f.clusterJSON.Topology.TiKV) > 0
		case proto.TidbComponentName:
			return len(f.clusterJSON.Topology.TiDB) > 0
		}
	}
	if f.clusterMeta != nil && f.clusterMeta.Topology != nil {
		switch name {
		case proto.PdComponentName:
			return len(f.clusterMeta.Topology.PDServers) > 0
		case proto.TikvComponentName:
			return len(f.clusterMeta.Topology.TiKVServers) > 0
		case proto.TidbComponentName:
			return len(f.clusterMeta.Topology.TiDBServers) > 0
		}
	}
	return false
}
