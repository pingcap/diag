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
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/diag/pkg/utils"
	perrs "github.com/pingcap/errors"
	dmspec "github.com/pingcap/tiup/components/dm/spec"
	"github.com/pingcap/tiup/pkg/cluster/spec"
)

// prepareArgsForTiUPCluster parses arguments and create output dir for tiup-cluster
// deployed tidb clusters
func (m *Manager) prepareArgsForTiUPCluster(
	opt *BaseOptions,
	cOpt *CollectOptions,
) (string, string, error) {
	// parse time range
	end, err := utils.ParseTime(opt.ScrapeEnd)
	if err != nil {
		return "", "", err
	}
	// if the begin time point is a minus integer, assume it as hour offset
	var start time.Time
	if offset, err := strconv.Atoi(opt.ScrapeBegin); err == nil && offset < 0 {
		start = end.Add(time.Hour * time.Duration(offset))
	} else {
		start, err = utils.ParseTime(opt.ScrapeBegin)
		if err != nil {
			return "", "", err
		}
	}

	// update time strings in setting to ensure all collectors work properly
	opt.ScrapeBegin = start.Format(time.RFC3339)
	opt.ScrapeEnd = end.Format(time.RFC3339)

	// show time range
	prompt := fmt.Sprintf(`Time range:
  %s - %s (Local)
  %s - %s (UTC)
  (total %.0f seconds)
`,
		color.HiYellowString(start.Local().Format(time.RFC3339)), color.HiYellowString(end.Local().Format(time.RFC3339)),
		color.HiYellowString(start.UTC().Format(time.RFC3339)), color.HiYellowString(end.UTC().Format(time.RFC3339)),
		end.Sub(start).Seconds(),
	)

	resultDir, err := m.getOutputDir(cOpt.Dir)
	return prompt, resultDir, err
}

// buildTopoForTiUPCluster creates an abstract topo from tiup-cluster metadata
func buildTopoForTiUPCluster(m *Manager, opt *BaseOptions) (*models.TiDBCluster, error) {
	exist, err := m.specManager.Exist(opt.Cluster)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, perrs.Errorf("cluster %s does not exist", opt.Cluster)
	}

	metadata := m.specManager.NewMetadata()
	err = m.specManager.Metadata(opt.Cluster, metadata)
	if err != nil {
		return nil, err
	}
	topo := metadata.GetTopology()

	if err != nil {
		return nil, err
	}
	opt.User = metadata.GetBaseMeta().User
	opt.SSH.IdentityFile = m.specManager.Path(opt.Cluster, "ssh", "id_rsa")

	// build the abstract topology
	cls := &models.TiDBCluster{
		Version: metadata.GetBaseMeta().Version,
		Attributes: map[string]interface{}{
			CollectModeTiUP: topo,
			"last_ops_ver":  metadata.GetBaseMeta().OpsVer,
		},
	}
	topo.IterInstance(func(ins spec.Instance) {

		switch ins.ComponentName() {
		case spec.ComponentPD:
			if cls.PD == nil {
				cls.PD = make([]*models.PDSpec, 0)
			}
			i := ins.(*spec.PDInstance).BaseInstance.InstanceSpec.(*spec.PDSpec)
			cls.PD = append(cls.PD, &models.PDSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       i.Host,
					Port:       i.GetMainPort(),
					StatusPort: i.ClientPort,
					SSHPort:    i.SSHPort,
					Attributes: map[string]interface{}{
						"name":       i.Name,
						"imported":   i.Imported,
						"patched":    i.Patched,
						"deploy_dir": i.DeployDir,
						"data_dir":   i.DataDir,
						"log_dir":    i.LogDir,
						"config":     i.Config,
						"os":         i.OS,
						"arch":       i.Arch,
					},
				},
			})
		case spec.ComponentTiKV:
			if cls.TiKV == nil {
				cls.TiKV = make([]*models.TiKVSpec, 0)
			}
			i := ins.(*spec.TiKVInstance).BaseInstance.InstanceSpec.(*spec.TiKVSpec)
			cls.TiKV = append(cls.TiKV, &models.TiKVSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       i.Host,
					Port:       i.GetMainPort(),
					StatusPort: i.StatusPort,
					SSHPort:    i.SSHPort,
					Attributes: map[string]interface{}{
						"imported":   i.Imported,
						"patched":    i.Patched,
						"deploy_dir": i.DeployDir,
						"data_dir":   i.DataDir,
						"log_dir":    i.LogDir,
						"config":     i.Config,
						"os":         i.OS,
						"arch":       i.Arch,
					},
				},
			})
		case spec.ComponentTiDB:
			if cls.TiDB == nil {
				cls.TiDB = make([]*models.TiDBSpec, 0)
			}
			i := ins.(*spec.TiDBInstance).BaseInstance.InstanceSpec.(*spec.TiDBSpec)
			cls.TiDB = append(cls.TiDB, &models.TiDBSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       i.Host,
					Port:       i.GetMainPort(),
					StatusPort: i.StatusPort,
					SSHPort:    i.SSHPort,
					Attributes: map[string]interface{}{
						"imported":   i.Imported,
						"patched":    i.Patched,
						"deploy_dir": i.DeployDir,
						"log_dir":    i.LogDir,
						"config":     i.Config,
						"os":         i.OS,
						"arch":       i.Arch,
					},
				},
			})
		case spec.ComponentTiFlash:
			if cls.TiFlash == nil {
				cls.TiFlash = make([]*models.TiFlashSpec, 0)
			}
			i := ins.(*spec.TiFlashInstance).BaseInstance.InstanceSpec.(*spec.TiFlashSpec)
			cls.TiFlash = append(cls.TiFlash, &models.TiFlashSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       i.Host,
					Port:       i.GetMainPort(),
					StatusPort: i.FlashProxyStatusPort,
					SSHPort:    i.SSHPort,
					Attributes: map[string]interface{}{
						"imported":   i.Imported,
						"patched":    i.Patched,
						"deploy_dir": i.DeployDir,
						"log_dir":    i.LogDir,
						"config":     i.Config,
						"os":         i.OS,
						"arch":       i.Arch,
					},
				},
			})
		case spec.ComponentPump:
			if cls.Pump == nil {
				cls.Pump = make([]*models.PumpSpec, 0)
			}
			i := ins.(*spec.PumpInstance).BaseInstance.InstanceSpec.(*spec.PumpSpec)
			cls.Pump = append(cls.Pump, &models.PumpSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       i.Host,
					Port:       i.GetMainPort(),
					StatusPort: 0,
					SSHPort:    i.SSHPort,
					Attributes: map[string]interface{}{
						"imported":   i.Imported,
						"patched":    i.Patched,
						"deploy_dir": i.DeployDir,
						"log_dir":    i.LogDir,
						"config":     i.Config,
						"os":         i.OS,
						"arch":       i.Arch,
					},
				},
			})
		case spec.ComponentDrainer:
			if cls.Drainer == nil {
				cls.Drainer = make([]*models.DrainerSpec, 0)
			}
			i := ins.(*spec.DrainerInstance).BaseInstance.InstanceSpec.(*spec.DrainerSpec)
			cls.Drainer = append(cls.Drainer, &models.DrainerSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       i.Host,
					Port:       i.GetMainPort(),
					StatusPort: 0,
					SSHPort:    i.SSHPort,
					Attributes: map[string]interface{}{
						"imported":   i.Imported,
						"patched":    i.Patched,
						"deploy_dir": i.DeployDir,
						"log_dir":    i.LogDir,
						"config":     i.Config,
						"os":         i.OS,
						"arch":       i.Arch,
						"ssh_port":   i.SSHPort,
					},
				},
			})
		case spec.ComponentCDC:
			if cls.TiCDC == nil {
				cls.TiCDC = make([]*models.TiCDCSpec, 0)
			}
			i := ins.(*spec.CDCInstance).BaseInstance.InstanceSpec.(*spec.CDCSpec)
			cls.TiCDC = append(cls.TiCDC, &models.TiCDCSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       i.Host,
					Port:       i.GetMainPort(),
					StatusPort: 0,
					SSHPort:    i.SSHPort,
					Attributes: map[string]interface{}{
						"imported":   i.Imported,
						"patched":    i.Patched,
						"deploy_dir": i.DeployDir,
						"log_dir":    i.LogDir,
						"config":     i.Config,
						"os":         i.OS,
						"arch":       i.Arch,
						"gc-ttl":     i.GCTTL,
						"tz":         i.TZ,
					},
				},
			})
		case spec.ComponentTiSpark:
			if cls.TiSpark == nil {
				cls.TiSpark = make([]*models.TiSparkCSpec, 0)
			}
			switch ins.Role() {
			case spec.RoleTiSparkMaster:
				i := ins.(*spec.TiSparkMasterInstance).BaseInstance.InstanceSpec.(*spec.TiSparkMasterSpec)
				cls.TiSpark = append(cls.TiSpark, &models.TiSparkCSpec{
					ComponentSpec: models.ComponentSpec{
						Host:       i.Host,
						Port:       i.GetMainPort(),
						StatusPort: 0,
						SSHPort:    i.SSHPort,
						Attributes: map[string]interface{}{
							"imported":   i.Imported,
							"patched":    i.Patched,
							"deploy_dir": i.DeployDir,
							"os":         i.OS,
							"arch":       i.Arch,
						},
					},
				})
			case spec.RoleTiSparkWorker:
				i := ins.(*spec.TiSparkWorkerInstance).BaseInstance.InstanceSpec.(*spec.TiSparkWorkerSpec)
				cls.TiSpark = append(cls.TiSpark, &models.TiSparkCSpec{
					ComponentSpec: models.ComponentSpec{
						Host:       i.Host,
						Port:       i.GetMainPort(),
						StatusPort: 0,
						SSHPort:    i.SSHPort,
						Attributes: map[string]interface{}{
							"imported":   i.Imported,
							"patched":    i.Patched,
							"deploy_dir": i.DeployDir,
							"os":         i.OS,
							"arch":       i.Arch,
						},
					},
				})
			}
		case dmspec.ComponentDMMaster:
			if cls.DMMaster == nil {
				cls.DMMaster = make([]*models.DMMasterSpec, 0)
			}
			i := ins.(*dmspec.MasterInstance).BaseInstance.InstanceSpec.(*dmspec.MasterSpec)
			cls.DMMaster = append(cls.DMMaster, &models.DMMasterSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       i.Host,
					Port:       i.GetMainPort(),
					StatusPort: 0,
					SSHPort:    i.SSHPort,
					Attributes: map[string]interface{}{
						"name":       i.Name,
						"imported":   i.Imported,
						"patched":    i.Patched,
						"deploy_dir": i.DeployDir,
						"data_dir":   i.DataDir,
						"log_dir":    i.LogDir,
						"config":     i.Config,
						"os":         i.OS,
						"arch":       i.Arch,
					},
				},
			})
		case dmspec.ComponentDMWorker:
			if cls.DMWorker == nil {
				cls.DMWorker = make([]*models.DMWorkerSpec, 0)
			}
			i := ins.(*dmspec.WorkerInstance).BaseInstance.InstanceSpec.(*dmspec.WorkerSpec)
			cls.DMWorker = append(cls.DMWorker, &models.DMWorkerSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       i.Host,
					Port:       i.GetMainPort(),
					StatusPort: 0,
					SSHPort:    i.SSHPort,
					Attributes: map[string]interface{}{
						"name":       i.Name,
						"imported":   i.Imported,
						"patched":    i.Patched,
						"deploy_dir": i.DeployDir,
						"data_dir":   i.DataDir,
						"log_dir":    i.LogDir,
						"config":     i.Config,
						"os":         i.OS,
						"arch":       i.Arch,
					},
				},
			})
		case spec.ComponentPrometheus:
			if cls.Monitors == nil {
				cls.Monitors = make([]*models.MonitorSpec, 0)
			}
			i := ins.(*spec.MonitorInstance).BaseInstance.InstanceSpec.(*spec.PrometheusSpec)
			cls.Monitors = append(cls.Monitors, &models.MonitorSpec{
				ComponentSpec: models.ComponentSpec{
					Host:       i.Host,
					Port:       i.GetMainPort(),
					StatusPort: 0,
					SSHPort:    i.SSHPort,
					Attributes: map[string]interface{}{
						"imported":   i.Imported,
						"patched":    i.Patched,
						"deploy_dir": i.DeployDir,
						"log_dir":    i.LogDir,
						"data_dir":   i.DataDir,
						"os":         i.OS,
						"arch":       i.Arch,
					},
				},
			})
		case spec.ComponentGrafana,
			spec.ComponentAlertmanager,
			spec.ComponentPushwaygate,
			spec.ComponentBlackboxExporter,
			spec.ComponentNodeExporter:
			// do nothing, skip
		default:
			fmt.Fprintf(os.Stderr,
				"instance %s is an unsupport/unecessary component (%s), skipped",
				ins.ID(), ins.ComponentName())
		}
	})

	return cls, nil
}
