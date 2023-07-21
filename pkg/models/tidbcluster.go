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

package models

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/set"
	mvccpb "go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// ComponentType are types of a component
type ComponentType string

// types for components
const (
	ComponentTypePD       ComponentType = "pd"
	ComponentTypeTiKV     ComponentType = "tikv"
	ComponentTypeTiDB     ComponentType = "tidb"
	ComponentTypeTiFlash  ComponentType = "tiflash"
	ComponentTypePump     ComponentType = "pump"
	ComponentTypeDrainer  ComponentType = "drainer"
	ComponentTypeTiCDC    ComponentType = "ticdc"
	ComponentTypeTiSpark  ComponentType = "tispark"
	ComponentTypeMonitor  ComponentType = "prometheus" // prometheus and/or ng-monitor
	ComponentTypeDMMaster ComponentType = "dm-master"
	ComponentTypeDMWorker ComponentType = "dm-worker"
)

// Component is the interface for any component
type Component interface {
	Type() ComponentType
	Host() string
	MainPort() int
	StatusPort() int
	SSHPort() int      // empty for tidb-operator
	ID() string        // host:port identifier
	StatusURL() string // the url to request for compoent status, without http/https scheme
	ConfigURL() string // the url to request for realtime configs, without http/https scheme
	Attributes() AttributeMap
}

// TiDBCluster is the abstract topology of a TiDB cluster
type TiDBCluster struct {
	Version    string          `json:"version"` // cluster version
	Namespace  string          `json:"namespace,omitempty"`
	Attributes AttributeMap    `json:"attributes,omitempty"`
	PD         []*PDSpec       `json:"pd,omitempty"` // PD not exist on DM cluster
	TiKV       []*TiKVSpec     `json:"tikv,omitempty"`
	TiDB       []*TiDBSpec     `json:"tidb,omitempty"`
	TiFlash    []*TiFlashSpec  `json:"tiflash,omitempty"`
	TiCDC      []*TiCDCSpec    `json:"ticdc,omitempty"`
	Pump       []*PumpSpec     `json:"pump,omitempty"`
	Drainer    []*DrainerSpec  `json:"drainer,omitempty"`
	TiSpark    []*TiSparkCSpec `json:"tispark,omitempty"`
	DMMaster   []*DMMasterSpec `json:"dm-master,omitempty"`
	DMWorker   []*DMWorkerSpec `json:"dm-worker,omitempty"`
	Monitors   []*MonitorSpec  `json:"monitors,omitempty"` // prometheus nodes
}

// Components list all available components in the cluster
func (c *TiDBCluster) Components() (comps []Component) {
	for _, i := range c.TiFlash {
		comps = append(comps, i)
	}
	for _, i := range c.PD {
		comps = append(comps, i)
	}
	for _, i := range c.TiKV {
		comps = append(comps, i)
	}
	for _, i := range c.Pump {
		comps = append(comps, i)
	}
	for _, i := range c.TiDB {
		comps = append(comps, i)
	}
	for _, i := range c.Drainer {
		comps = append(comps, i)
	}
	for _, i := range c.TiCDC {
		comps = append(comps, i)
	}
	for _, i := range c.TiSpark {
		comps = append(comps, i)
	}
	for _, i := range c.DMMaster {
		comps = append(comps, i)
	}
	for _, i := range c.DMWorker {
		comps = append(comps, i)
	}
	for _, i := range c.Monitors {
		comps = append(comps, i)
	}
	return
}

// ComponentSpec is the definition of general component
type ComponentSpec struct {
	Host       string       `json:"host"`
	Port       int          `json:"port"`
	StatusPort int          `json:"status_port"`
	SSHPort    int          `json:"ssh_port,omitempty"`
	Attributes AttributeMap `json:"attributes,omitempty"`
}

// MonitorSpec is the spec needed for monitoring nodes
type MonitorSpec struct {
	ComponentSpec `json:",inline"`
	Endpoint      string `json:"endpoint,omitempty"`    // the metric endpoint, normally prometheus address
	NGEndpoint    string `json:"ng_endpoint,omitempty"` // the endpoint of NG-monitor, might be empty
}

// Type implements Component interface
func (s *MonitorSpec) Type() ComponentType { return ComponentTypeMonitor }

// Host implements Component interface
func (s *MonitorSpec) Host() string { return s.ComponentSpec.Host }

// Domain implements Component interface
func (s *MonitorSpec) Domain() string {
	if domain, ok := s.Attributes()["domain"].(string); ok {
		return domain
	}
	return ""
}

// MainPort implements Component interface
func (s *MonitorSpec) MainPort() int { return s.ComponentSpec.Port }

// StatusPort implements Component interface
func (s *MonitorSpec) StatusPort() int { return s.ComponentSpec.StatusPort }

// SSHPort implements Component interface
func (s *MonitorSpec) SSHPort() int { return s.ComponentSpec.SSHPort }

// StatusURL implements Component interface
func (s *MonitorSpec) StatusURL() string {
	return ""
}

// ConfigURL implements Component interface
func (s *MonitorSpec) ConfigURL() string {
	return s.StatusURL()
}

// ID implements Component interface
func (s *MonitorSpec) ID() string { return fmt.Sprintf("%s:%d", s.Host(), s.MainPort()) }

// Attributes implements Component interface
func (s *MonitorSpec) Attributes() AttributeMap { return s.ComponentSpec.Attributes }

// PDSpec represent PD nodes
type PDSpec struct {
	ComponentSpec `json:",inline"`
}

// Type implements Component interface
func (s *PDSpec) Type() ComponentType { return ComponentTypePD }

// Host implements Component interface
func (s *PDSpec) Host() string { return s.ComponentSpec.Host }

// Domain implements Component interface
func (s *PDSpec) Domain() string {
	if domain, ok := s.Attributes()["domain"].(string); ok {
		return domain
	}
	return ""
}

// MainPort implements Component interface
func (s *PDSpec) MainPort() int { return s.ComponentSpec.Port }

// StatusPort implements Component interface
func (s *PDSpec) StatusPort() int { return s.ComponentSpec.StatusPort }

// SSHPort implements Component interface
func (s *PDSpec) SSHPort() int { return s.ComponentSpec.SSHPort }

// StatusURL implements Component interface
func (s *PDSpec) StatusURL() string {
	if s.Domain() != "" {
		return fmt.Sprintf("%s:%d", s.Domain(), s.StatusPort())
	}
	return fmt.Sprintf("%s:%d", s.Host(), s.StatusPort())
}

// ConfigURL implements Component interface
func (s *PDSpec) ConfigURL() string {
	return fmt.Sprintf("%s/pd/api/v1/config", s.StatusURL())
}

// ID implements Component interface
func (s *PDSpec) ID() string { return fmt.Sprintf("%s:%d", s.Host(), s.MainPort()) }

// Attributes implements Component interface
func (s *PDSpec) Attributes() AttributeMap { return s.ComponentSpec.Attributes }

// TiKVSpec represent PD nodes
type TiKVSpec struct {
	ComponentSpec `json:",inline"`
}

// Type implements Component interface
func (s *TiKVSpec) Type() ComponentType { return ComponentTypeTiKV }

// Host implements Component interface
func (s *TiKVSpec) Host() string { return s.ComponentSpec.Host }

// Domain implements Component interface
func (s *TiKVSpec) Domain() string {
	if domain, ok := s.Attributes()["domain"].(string); ok {
		return domain
	}
	return ""
}

// MainPort implements Component interface
func (s *TiKVSpec) MainPort() int { return s.ComponentSpec.Port }

// StatusPort implements Component interface
func (s *TiKVSpec) StatusPort() int { return s.ComponentSpec.StatusPort }

// SSHPort implements Component interface
func (s *TiKVSpec) SSHPort() int { return s.ComponentSpec.SSHPort }

// StatusURL implements Component interface
func (s *TiKVSpec) StatusURL() string {
	if s.Domain() != "" {
		return fmt.Sprintf("%s:%d", s.Domain(), s.StatusPort())
	}
	return fmt.Sprintf("%s:%d", s.Host(), s.StatusPort())
}

// ConfigURL implements Component interface
func (s *TiKVSpec) ConfigURL() string {
	return fmt.Sprintf("%s/config", s.StatusURL())
}

// ID implements Component interface
func (s *TiKVSpec) ID() string { return fmt.Sprintf("%s:%d", s.Host(), s.MainPort()) }

// Attributes implements Component interface
func (s *TiKVSpec) Attributes() AttributeMap { return s.ComponentSpec.Attributes }

// TiDBSpec represent PD nodes
type TiDBSpec struct {
	ComponentSpec `json:",inline"`
}

// Type implements Component interface
func (s *TiDBSpec) Type() ComponentType { return ComponentTypeTiDB }

// Host implements Component interface
func (s *TiDBSpec) Host() string { return s.ComponentSpec.Host }

// Domain implements Component interface
func (s *TiDBSpec) Domain() string {
	if domain, ok := s.Attributes()["domain"].(string); ok {
		return domain
	}
	return ""
}

// MainPort implements Component interface
func (s *TiDBSpec) MainPort() int { return s.ComponentSpec.Port }

// StatusPort implements Component interface
func (s *TiDBSpec) StatusPort() int { return s.ComponentSpec.StatusPort }

// SSHPort implements Component interface
func (s *TiDBSpec) SSHPort() int { return s.ComponentSpec.SSHPort }

// StatusURL implements Component interface
func (s *TiDBSpec) StatusURL() string {
	if s.Domain() != "" {
		return fmt.Sprintf("%s:%d", s.Domain(), s.StatusPort())
	}
	return fmt.Sprintf("%s:%d", s.Host(), s.StatusPort())
}

// ConfigURL implements Component interface
func (s *TiDBSpec) ConfigURL() string {
	return fmt.Sprintf("%s/config", s.StatusURL())
}

// ID implements Component interface
func (s *TiDBSpec) ID() string { return fmt.Sprintf("%s:%d", s.Host(), s.MainPort()) }

// Attributes implements Component interface
func (s *TiDBSpec) Attributes() AttributeMap { return s.ComponentSpec.Attributes }

// TiFlashSpec represent PD nodes
type TiFlashSpec struct {
	ComponentSpec `json:",inline"`
}

// Type implements Component interface
func (s *TiFlashSpec) Type() ComponentType { return ComponentTypeTiFlash }

// Host implements Component interface
func (s *TiFlashSpec) Host() string { return s.ComponentSpec.Host }

// Domain implements Component interface
func (s *TiFlashSpec) Domain() string {
	if domain, ok := s.Attributes()["domain"].(string); ok {
		return domain
	}
	return ""
}

// MainPort implements Component interface
func (s *TiFlashSpec) MainPort() int { return s.ComponentSpec.Port }

// StatusPort implements Component interface
func (s *TiFlashSpec) StatusPort() int { return s.ComponentSpec.StatusPort }

// SSHPort implements Component interface
func (s *TiFlashSpec) SSHPort() int { return s.ComponentSpec.SSHPort }

// StatusURL implements Component interface
func (s *TiFlashSpec) StatusURL() string {
	if s.Domain() != "" {
		return fmt.Sprintf("%s:%d", s.Domain(), s.StatusPort())
	}
	return fmt.Sprintf("%s:%d", s.Host(), s.StatusPort())
}

// ConfigURL implements Component interface
func (s *TiFlashSpec) ConfigURL() string {
	return fmt.Sprintf("%s/config", s.StatusURL())
}

// ID implements Component interface
func (s *TiFlashSpec) ID() string { return fmt.Sprintf("%s:%d", s.Host(), s.MainPort()) }

// Attributes implements Component interface
func (s *TiFlashSpec) Attributes() AttributeMap { return s.ComponentSpec.Attributes }

// PumpSpec represent PD nodes
type PumpSpec struct {
	ComponentSpec `json:",inline"`
}

// Type implements Component interface
func (s *PumpSpec) Type() ComponentType { return ComponentTypePump }

// Host implements Component interface
func (s *PumpSpec) Host() string { return s.ComponentSpec.Host }

// Domain implements Component interface
func (s *PumpSpec) Domain() string {
	if domain, ok := s.Attributes()["domain"].(string); ok {
		return domain
	}
	return ""
}

// MainPort implements Component interface
func (s *PumpSpec) MainPort() int { return s.ComponentSpec.Port }

// StatusPort implements Component interface
func (s *PumpSpec) StatusPort() int { return s.ComponentSpec.StatusPort }

// SSHPort implements Component interface
func (s *PumpSpec) SSHPort() int { return s.ComponentSpec.SSHPort }

// StatusURL implements Component interface
func (s *PumpSpec) StatusURL() string {
	return ""
}

// ConfigURL implements Component interface
func (s *PumpSpec) ConfigURL() string {
	return s.StatusURL()
}

// ID implements Component interface
func (s *PumpSpec) ID() string { return fmt.Sprintf("%s:%d", s.Host(), s.MainPort()) }

// Attributes implements Component interface
func (s *PumpSpec) Attributes() AttributeMap { return s.ComponentSpec.Attributes }

// DrainerSpec represent PD nodes
type DrainerSpec struct {
	ComponentSpec `json:",inline"`
}

// Type implements Component interface
func (s *DrainerSpec) Type() ComponentType { return ComponentTypeDrainer }

// Host implements Component interface
func (s *DrainerSpec) Host() string { return s.ComponentSpec.Host }

// Domain implements Component interface
func (s *DrainerSpec) Domain() string {
	if domain, ok := s.Attributes()["domain"].(string); ok {
		return domain
	}
	return ""
}

// MainPort implements Component interface
func (s *DrainerSpec) MainPort() int { return s.ComponentSpec.Port }

// StatusPort implements Component interface
func (s *DrainerSpec) StatusPort() int { return s.ComponentSpec.StatusPort }

// SSHPort implements Component interface
func (s *DrainerSpec) SSHPort() int { return s.ComponentSpec.SSHPort }

// StatusURL implements Component interface
func (s *DrainerSpec) StatusURL() string {
	return ""
}

// ConfigURL implements Component interface
func (s *DrainerSpec) ConfigURL() string {
	return s.StatusURL()
}

// ID implements Component interface
func (s *DrainerSpec) ID() string { return fmt.Sprintf("%s:%d", s.Host(), s.MainPort()) }

// Attributes implements Component interface
func (s *DrainerSpec) Attributes() AttributeMap { return s.ComponentSpec.Attributes }

// TiCDCSpec represent PD nodes
type TiCDCSpec struct {
	ComponentSpec `json:",inline"`
}

// Type implements Component interface
func (s *TiCDCSpec) Type() ComponentType { return ComponentTypeTiCDC }

// Host implements Component interface
func (s *TiCDCSpec) Host() string { return s.ComponentSpec.Host }

// Domain implements Component interface
func (s *TiCDCSpec) Domain() string {
	if domain, ok := s.Attributes()["domain"].(string); ok {
		return domain
	}
	return ""
}

// MainPort implements Component interface
func (s *TiCDCSpec) MainPort() int { return s.ComponentSpec.Port }

// StatusPort implements Component interface
func (s *TiCDCSpec) StatusPort() int { return s.ComponentSpec.StatusPort }

// SSHPort implements Component interface
func (s *TiCDCSpec) SSHPort() int { return s.ComponentSpec.SSHPort }

// StatusURL implements Component interface
func (s *TiCDCSpec) StatusURL() string {
	if s.Domain() != "" {
		return fmt.Sprintf("%s:%d", s.Domain(), s.MainPort())
	}
	return fmt.Sprintf("%s:%d", s.Host(), s.MainPort())
}

// ConfigURL implements Component interface
func (s *TiCDCSpec) ConfigURL() string {
	return s.StatusURL()
}

// ID implements Component interface
func (s *TiCDCSpec) ID() string { return fmt.Sprintf("%s:%d", s.Host(), s.MainPort()) }

// Attributes implements Component interface
func (s *TiCDCSpec) Attributes() AttributeMap { return s.ComponentSpec.Attributes }

// TiSparkCSpec represent PD nodes
type TiSparkCSpec struct {
	ComponentSpec `json:",inline"`
	Master        bool `json:"master"`
}

// Type implements Component interface
func (s *TiSparkCSpec) Type() ComponentType { return ComponentTypeTiSpark }

// Host implements Component interface
func (s *TiSparkCSpec) Host() string { return s.ComponentSpec.Host }

// Domain implements Component interface
func (s *TiSparkCSpec) Domain() string {
	if domain, ok := s.Attributes()["domain"].(string); ok {
		return domain
	}
	return ""
}

// MainPort implements Component interface
func (s *TiSparkCSpec) MainPort() int { return s.ComponentSpec.Port }

// StatusPort implements Component interface
func (s *TiSparkCSpec) StatusPort() int { return s.ComponentSpec.StatusPort }

// SSHPort implements Component interface
func (s *TiSparkCSpec) SSHPort() int { return s.ComponentSpec.SSHPort }

// StatusURL implements Component interface
func (s *TiSparkCSpec) StatusURL() string {
	return ""
}

// ConfigURL implements Component interface
func (s *TiSparkCSpec) ConfigURL() string {
	return s.StatusURL()
}

// ID implements Component interface
func (s *TiSparkCSpec) ID() string { return fmt.Sprintf("%s:%d", s.Host(), s.MainPort()) }

// Attributes implements Component interface
func (s *TiSparkCSpec) Attributes() AttributeMap { return s.ComponentSpec.Attributes }

// IsMster checks if the node is a TiSpark master
func (s *TiSparkCSpec) IsMaster() bool { return s.Master }

// DMMaterSpec represent PD nodes
type DMMasterSpec struct {
	ComponentSpec `json:",inline"`
}

// Type implements Component interface
func (s *DMMasterSpec) Type() ComponentType { return ComponentTypeDMMaster }

// Host implements Component interface
func (s *DMMasterSpec) Host() string { return s.ComponentSpec.Host }

// Domain implements Component interface
func (s *DMMasterSpec) Domain() string {
	if domain, ok := s.Attributes()["domain"].(string); ok {
		return domain
	}
	return ""
}

// MainPort implements Component interface
func (s *DMMasterSpec) MainPort() int { return s.ComponentSpec.Port }

// StatusPort implements Component interface
func (s *DMMasterSpec) StatusPort() int { return s.ComponentSpec.StatusPort }

// SSHPort implements Component interface
func (s *DMMasterSpec) SSHPort() int { return s.ComponentSpec.SSHPort }

// StatusURL implements Component interface
func (s *DMMasterSpec) StatusURL() string {
	return ""
}

// ConfigURL implements Component interface
func (s *DMMasterSpec) ConfigURL() string {
	return s.StatusURL()
}

// ID implements Component interface
func (s *DMMasterSpec) ID() string { return fmt.Sprintf("%s:%d", s.Host(), s.MainPort()) }

// Attributes implements Component interface
func (s *DMMasterSpec) Attributes() AttributeMap { return s.ComponentSpec.Attributes }

// DMWorkerSpec represent PD nodes
type DMWorkerSpec struct {
	ComponentSpec `json:",inline"`
}

// Type implements Component interface
func (s *DMWorkerSpec) Type() ComponentType { return ComponentTypeDMWorker }

// Host implements Component interface
func (s *DMWorkerSpec) Host() string { return s.ComponentSpec.Host }

// Domain implements Component interface
func (s *DMWorkerSpec) Domain() string {
	if domain, ok := s.Attributes()["domain"].(string); ok {
		return domain
	}
	return ""
}

// MainPort implements Component interface
func (s *DMWorkerSpec) MainPort() int { return s.ComponentSpec.Port }

// StatusPort implements Component interface
func (s *DMWorkerSpec) StatusPort() int { return s.ComponentSpec.StatusPort }

// SSHPort implements Component interface
func (s *DMWorkerSpec) SSHPort() int { return s.ComponentSpec.SSHPort }

// StatusURL implements Component interface
func (s *DMWorkerSpec) StatusURL() string {
	if s.Domain() != "" {
		return fmt.Sprintf("%s:%d", s.Domain(), s.StatusPort())
	}
	return fmt.Sprintf("%s:%d", s.Host(), s.StatusPort())
}

// ConfigURL implements Component interface
func (s *DMWorkerSpec) ConfigURL() string {
	return fmt.Sprintf("%s/config", s.StatusURL())
}

// ID implements Component interface
func (s *DMWorkerSpec) ID() string { return fmt.Sprintf("%s:%d", s.Host(), s.MainPort()) }

// Attributes implements Component interface
func (s *DMWorkerSpec) Attributes() AttributeMap { return s.ComponentSpec.Attributes }

// FilterComponent filter components by set
func FilterComponent(comps []Component, components set.StringSet) (res []Component) {
	if len(components) == 0 {
		res = comps
		return
	}

	for _, c := range comps {
		switch c.Type() {
		case ComponentTypeTiSpark: // tispark is not available in tidb-operator
			rm := spec.RoleTiSparkMaster
			rs := spec.RoleTiSparkWorker
			if !components.Exist(rm) && !components.Exist(rs) {
				continue
			}
		default:
			role := string(c.Type())
			if !components.Exist(role) {
				continue
			}
		}

		res = append(res, c)
	}

	return
}

// FilterInstance filter instances by set
func FilterInstance(instances []Component, nodes set.StringSet) (res []Component) {
	if len(nodes) == 0 {
		res = instances
		return
	}

	for _, c := range instances {
		if !nodes.Exist(c.ID()) {
			continue
		}
		res = append(res, c)
	}

	return
}

// GetEtcdClient loads EtcdClient of current cluster
func (c *TiDBCluster) GetEtcdClient(tlsCfg *tls.Config) (*clientv3.Client, error) {
	var pdList []string

	for _, pd := range c.PD {
		if pd.Domain() != "" {
			pdList = append(pdList, fmt.Sprintf("%s:%d", pd.Domain(), pd.StatusPort()))
		}
		pdList = append(pdList, fmt.Sprintf("%s:%d", pd.Host(), pd.StatusPort()))
	}

	return clientv3.New(clientv3.Config{
		Endpoints: pdList,
		TLS:       tlsCfg,
	})
}

const ticdcEtcdKeyBase string = "/tidb/cdc"

// getAllCDCInfo get all keys created by CDC
func (c *TiDBCluster) GetAllCDCInfo(ctx context.Context, timeout time.Duration, tlsCfg *tls.Config) ([]*mvccpb.KeyValue, error) {
	if timeout < time.Second {
		timeout = 5 * time.Second
	}

	etcdClient, err := c.GetEtcdClient(tlsCfg)
	if err != nil {
		return nil, err
	}

	ctx, f := context.WithTimeout(ctx, timeout)
	defer f()
	resp, err := etcdClient.Get(ctx, ticdcEtcdKeyBase, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	return resp.Kvs, nil
}
