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
	"encoding/json"
	"os"
	"path"

	"github.com/pingcap/diag/checker/config"
	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/diag/collector"
	"github.com/pingcap/log"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Fetcher interface {
	FetchData(rules *config.RuleSpec) (*proto.SourceDataV2, proto.RuleSet, error)
}

// TODO: move to consts pkg later
const (
	clusterMetaFileName = "meta.yaml"
	clusterJSONFileName = "cluster.json"
)

// FileFetcher load all needed data from file
type FileFetcher struct {
	dataDirPath string // dataDirPath point to a folder
}

func NewFileFetcher(dataDirPath string) (*FileFetcher, error) {
	ff := FileFetcher{
		dataDirPath: dataDirPath,
	}
	return &ff, nil
}

// FetchData retrieve config data from file path, and filter rules by component version
// dataPath is the path to the top folder which contain the data collected by diag collect.
func (f *FileFetcher) FetchData(rules *config.RuleSpec) (*proto.SourceDataV2, proto.RuleSet, error) {
	meta := &spec.ClusterMeta{}
	clusterJSON := &collector.ClusterJSON{}
	{
		// decode meta.yaml
		bs, err := os.ReadFile(f.genMetaConfigPath())
		if err != nil {
			return nil, nil, err
		}
		if err := yaml.Unmarshal(bs, meta); err != nil {
			log.Error(err.Error())
			return nil, nil, err
		}
	}
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
	rSet, err := rules.FilterOnVersion(meta.Version)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, err
	}
	sourceData := &proto.SourceDataV2{
		ClusterInfo: clusterJSON,
		NodesData:   make(map[string][]proto.Config),
		TidbVersion: meta.Version,
	}
	// decode config.json for related component
	for name := range rSet {
		switch name {
		case proto.PdComponentName:
			for _, spec := range meta.Topology.PDServers {
				// todo add no data
				cfgPath := path.Join(f.dataDirPath, spec.Host, spec.DeployDir, "conf", "config.json")
				bs, err := os.ReadFile(cfgPath)
				cfg := proto.NewPdConfigData()
				if err != nil {
					cfg.PdConfig = nil // skip error
				} else {
					if err := json.Unmarshal(bs, cfg); err != nil {
						logrus.Error(err)
						return nil, nil, err
					}
				}
				cfg.Port = spec.ClientPort
				cfg.Host = spec.Host
				f.Append(sourceData, cfg, proto.PdComponentName)
			}
		case proto.TikvComponentName:
			for _, spec := range meta.Topology.TiKVServers {
				cfgPath := path.Join(f.dataDirPath, spec.Host, spec.DeployDir, "conf", "config.json")
				bs, err := os.ReadFile(cfgPath)
				cfg := proto.NewTikvConfigData()
				if err != nil {
					cfg.TikvConfig = nil // skip error
				} else {
					if err := json.Unmarshal(bs, cfg); err != nil {
						logrus.Error(err)
						return nil, nil, err
					}
					cfg.LoadClusterMeta(meta)
				}
				cfg.Host = spec.Host
				cfg.Port = spec.Port
				f.Append(sourceData, cfg, proto.TikvComponentName)
			}
		case proto.TidbComponentName:
			for _, spec := range meta.Topology.TiDBServers {
				cfgPath := path.Join(f.dataDirPath, spec.Host, spec.DeployDir, "conf", "config.json")
				bs, err := os.ReadFile(cfgPath)
				cfg := proto.NewTidbConfigData()
				if err != nil {
					cfg.TidbConfig = nil
				} else {
					if err := json.Unmarshal(bs, cfg); err != nil {
						logrus.Error(err)
						return nil, nil, err
					}
				}
				cfg.Host = spec.Host
				cfg.Port = spec.Port
				f.Append(sourceData, cfg, proto.TidbComponentName)
			}
		default:
		}
	}
	return sourceData, rSet, nil
}

func (f *FileFetcher) Append(sourceData *proto.SourceDataV2, cfg proto.Config, component string) {
	if n, ok := sourceData.NodesData[component]; ok {
		n = append(n, cfg)
		sourceData.NodesData[component] = n
	} else {
		n = []proto.Config{cfg}
		sourceData.NodesData[component] = n
	}
}

func (f *FileFetcher) genMetaConfigPath() string {
	return path.Join(f.dataDirPath, clusterMetaFileName)
}

func (f *FileFetcher) genClusterJSONPath() string {
	return path.Join(f.dataDirPath, clusterJSONFileName)
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
