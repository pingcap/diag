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
	"encoding/json"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/diag/checker/config"
	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/diag/collector"
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/errors"
	"github.com/pingcap/log"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Fetcher interface {
	FetchData(rules *config.RuleSpec) (*proto.SourceDataV2, proto.RuleSet, error)
}

const (
	ConfigFlag CheckFlag = 1 << iota
	PerformanceFlag
)

type CheckFlag int

func (cf CheckFlag) checkConfig() bool {
	return cf&ConfigFlag > 0
}

func (cf CheckFlag) checkPerformance() bool {
	return cf&PerformanceFlag > 0
}

// FileFetcher load all needed data from file
type FileFetcher struct {
	dataDirPath string // dataDirPath point to a folder
	checkFlag   CheckFlag
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
	meta := &models.TiDBCluster{}
	clusterJSON := &collector.ClusterJSON{}
	{
		// decode cluster.json
		bs, err := os.ReadFile(f.genClusterJSONPath())
		if err != nil {
			return nil, nil, err
		}
		if err := json.Unmarshal(bs, clusterJSON); err != nil {
			log.Error(err.Error())
			return nil, nil, err
		}
	}

	{
		// decode topology.json
		bs, err := os.ReadFile(f.genMetaConfigPath())
		if err != nil {
			return nil, nil, err
		}
		if err := yaml.Unmarshal(bs, meta); err != nil {
			log.Error(err.Error())
			return nil, nil, err
		}
	}
	filterFunc := func(item config.RuleItem) (bool, error) {
		// filter on version
		ok, err := item.Version.Contain(meta.Version)
		if err != nil {
			return false, err
		} else if !ok {
			return false, nil
		}

		// filter on check flag
		switch item.NameStruct {
		case proto.PdComponentName, proto.TikvComponentName, proto.TidbComponentName, proto.TiflashComponentName:
			return f.checkFlag.checkConfig(), nil
		case proto.PerformanceDashboardComponentName:
			return f.checkFlag.checkPerformance(), nil
		default:
			return false, nil
		}
	}
	// rSet contain all rules match the cluster version
	rSet, err := rules.FilterOn(filterFunc)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, err
	}
	sourceData := &proto.SourceDataV2{
		ClusterInfo:   clusterJSON,
		NodesData:     make(map[string][]proto.Config),
		TidbVersion:   meta.Version,
		DashboardData: &proto.DashboardData{},
	}
	ctx := context.Background()
	// decode config.json for related config component
	if f.checkFlag.checkConfig() {
		if err := f.loadRealTimeConfig(ctx, sourceData, meta, rSet); err != nil {
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

		if err := f.loadSlowLog(ctx, sourceData, meta); err != nil {
			return nil, nil, err
		}

		//{
		//	reader, err := os.Open(f.genInfoSchemaCSVPath("avg_process_time_by_plan.csv"))
		//	if err != nil {
		//		return nil, nil, err
		//	}
		//	defer reader.Close()
		//	data, err := f.loadSlowPlanData(reader)
		//	if err != nil {
		//		return nil, nil, err
		//	}
		//	sourceData.DashboardData.ExecutionPlanInfoList = data
		//}
		//{
		//	reader, err := os.Open(f.genInfoSchemaCSVPath("key_old_version_plan.csv"))
		//	if err != nil {
		//		return nil, nil, err
		//	}
		//	defer reader.Close()
		//	data, err := f.loadDigest(reader)
		//	if err != nil {
		//		return nil, nil, err
		//	}
		//	sourceData.DashboardData.OldVersionProcesskeyCount.DataList = data
		//	sourceData.DashboardData.OldVersionProcesskeyCount.Count = len(data)
		//}
		//{
		//	reader, err := os.Open(f.genInfoSchemaCSVPath("skip_toomany_keys_plan.csv"))
		//	if err != nil {
		//		return nil, nil, err
		//	}
		//	defer reader.Close()
		//	data, err := f.loadDigest(reader)
		//	if err != nil {
		//		return nil, nil, err
		//	}
		//	sourceData.DashboardData.TombStoneStatistics.DataList = data
		//	sourceData.DashboardData.TombStoneStatistics.Count = len(data)
		//}
	}

	return sourceData, rSet, nil
}

// sourceData will be updated
func (f *FileFetcher) loadRealTimeConfig(ctx context.Context, sourceData *proto.SourceDataV2, meta *models.TiDBCluster, rSet proto.RuleSet) error {
	nameStructs := rSet.GetNameStructs()
	for name := range nameStructs {
		switch name {
		case proto.PdComponentName:
			for _, spec := range meta.PD {
				// todo add no data
				cfgPath := path.Join(f.dataDirPath, spec.Host(), "conf", "config.json")
				if deployDir, ok := spec.Attributes()["deploy_dir"]; ok {
					cfgPath = path.Join(f.dataDirPath, spec.Host(), deployDir.(string), "conf", "config.json")
				}
				bs, err := os.ReadFile(cfgPath)
				cfg := proto.NewPdConfigData()
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
			for _, spec := range meta.TiKV {
				cfgPath := path.Join(f.dataDirPath, spec.Host(), "conf", "config.json")
				if deployDir, ok := spec.Attributes()["deploy_dir"]; ok {
					cfgPath = path.Join(f.dataDirPath, spec.Host(), deployDir.(string), "conf", "config.json")
				}
				bs, err := os.ReadFile(cfgPath)
				cfg := proto.NewTikvConfigData()
				if err != nil {
					cfg.TikvConfig = nil // skip error
				} else {
					if err := json.Unmarshal(bs, cfg); err != nil {
						logrus.Error(err)
						return err
					}
					cfg.LoadClusterMeta(meta)
				}
				cfg.Host = spec.Host()
				cfg.Port = spec.MainPort()
				sourceData.AppendConfig(cfg, proto.TikvComponentName)
			}
		case proto.TidbComponentName:
			for _, spec := range meta.TiDB {
				cfgPath := path.Join(f.dataDirPath, spec.Host(), "conf", "config.json")
				if deployDir, ok := spec.Attributes()["deploy_dir"]; ok {
					cfgPath = path.Join(f.dataDirPath, spec.Host(), deployDir.(string), "conf", "config.json")
				}
				bs, err := os.ReadFile(cfgPath)
				cfg := proto.NewTidbConfigData()
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
func (f *FileFetcher) loadSlowLog(ctx context.Context, sourceData *proto.SourceDataV2, meta *models.TiDBCluster) (err error) {
	header := []string{"Time", "Digest", "Plan_digest", "Process_time", "Process_keys", "Rocksdb_delete_skipped_count", "Total_keys"}
	idxLookUp := NewIdxLookup(header)
	avgProcessTimePlanAcc := avgProcessTimePlanAccumulator{
		idxLookUp: idxLookUp,
		data:      make(map[string]map[string]*execTimeInfo),
	}
	scanOldVersionPlanAcc := scanOldVersionPlanAccumulator{
		idxLookUp: idxLookUp,
		data:      make(map[string]map[string]struct{}),
	}
	skipDeletedCntPlanAcc := skipDeletedCntPlanAccumulator{
		idxLookUp: idxLookUp,
		data:      make(map[string]map[string]struct{}),
	}
	closers := make([]io.Closer, 0)
	for _, spec := range meta.TiDB {
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
		data, err := retriever.retrieve(ctx)
		if err != nil {
			log.Warn("retrieve slow log failed", zap.Error(err))
			continue
		}
		for _, row := range data {
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
	for _, c := range closers {
		c.Close()
	}
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
		lastTime, err := time.Parse("2006-01-02 15:04:05", record[idxLookUp["last_time"]])
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
	return path.Join(f.dataDirPath, collector.FileNameClusterAbstractTopo)
}

func (f *FileFetcher) genClusterJSONPath() string {
	return path.Join(f.dataDirPath, collector.FileNameClusterJSON)
}

func (f *FileFetcher) genInfoSchemaCSVPath(fileName string) string {
	return path.Join(f.dataDirPath, collector.DirNameInfoSchema, fileName)
}

func (f *FileFetcher) GetComponent(meta *spec.ClusterMeta) []string {
	components := make([]string, 0)
	if len(meta.Topology.PDServers) != 0 {
		components = append(components, proto.PdComponentName)
	}
	if len(meta.Topology.TiDBServers) != 0 {
		components = append(components, proto.TidbComponentName)
	}
	if len(meta.Topology.TiKVServers) != 0 {
		components = append(components, proto.TikvComponentName)
	}
	if len(meta.Topology.TiFlashServers) != 0 {
		components = append(components, proto.TiflashComponentName)
	}
	return components
}
