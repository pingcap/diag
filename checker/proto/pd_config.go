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

package proto

import (
	"reflect"
	"strings"

	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/tiup/pkg/cluster/api/typeutil"
	"go.uber.org/zap"
)

// StoreLimitConfig is a config about scheduling rate limit of different types for a store.
type StoreLimitConfig struct {
	AddPeer    float64 `toml:"add-peer" json:"add-peer"`
	RemovePeer float64 `toml:"remove-peer" json:"remove-peer"`
}

// SchedulerConfigs is a slice of customized scheduler configuration.
type SchedulerConfigs []SchedulerConfig

// ScheduleConfig is the schedule configuration.
type ScheduleConfig struct {
	// If the snapshot count of one store is greater than this value,
	// it will never be used as a source or target store.
	MaxSnapshotCount    uint64 `toml:"max-snapshot-count" json:"max-snapshot-count"`
	MaxPendingPeerCount uint64 `toml:"max-pending-peer-count" json:"max-pending-peer-count"`
	// If both the size of region is smaller than MaxMergeRegionSize
	// and the number of rows in region is smaller than MaxMergeRegionKeys,
	// it will try to merge with adjacent regions.
	MaxMergeRegionSize uint64 `toml:"max-merge-region-size" json:"max-merge-region-size"`
	MaxMergeRegionKeys uint64 `toml:"max-merge-region-keys" json:"max-merge-region-keys"`
	// SplitMergeInterval is the minimum interval time to permit merge after split.
	SplitMergeInterval typeutil.Duration `toml:"split-merge-interval" json:"split-merge-interval"`
	// EnableOneWayMerge is the option to enable one way merge. This means a Region can only be merged into the next region of it.
	EnableOneWayMerge bool `toml:"enable-one-way-merge" json:"enable-one-way-merge,string"`
	// EnableCrossTableMerge is the option to enable cross table merge. This means two Regions can be merged with different table IDs.
	// This option only works when key type is "table".
	EnableCrossTableMerge bool `toml:"enable-cross-table-merge" json:"enable-cross-table-merge,string"`
	// PatrolRegionInterval is the interval for scanning region during patrol.
	PatrolRegionInterval typeutil.Duration `toml:"patrol-region-interval" json:"patrol-region-interval"`
	// MaxStoreDownTime is the max duration after which
	// a store will be considered to be down if it hasn't reported heartbeats.
	MaxStoreDownTime typeutil.Duration `toml:"max-store-down-time" json:"max-store-down-time"`
	// LeaderScheduleLimit is the max coexist leader schedules.
	LeaderScheduleLimit uint64 `toml:"leader-schedule-limit" json:"leader-schedule-limit"`
	// LeaderSchedulePolicy is the option to balance leader, there are some policies supported: ["count", "size"], default: "count"
	LeaderSchedulePolicy string `toml:"leader-schedule-policy" json:"leader-schedule-policy"`
	// RegionScheduleLimit is the max coexist region schedules.
	RegionScheduleLimit uint64 `toml:"region-schedule-limit" json:"region-schedule-limit"`
	// ReplicaScheduleLimit is the max coexist replica schedules.
	ReplicaScheduleLimit uint64 `toml:"replica-schedule-limit" json:"replica-schedule-limit"`
	// MergeScheduleLimit is the max coexist merge schedules.
	MergeScheduleLimit uint64 `toml:"merge-schedule-limit" json:"merge-schedule-limit"`
	// HotRegionScheduleLimit is the max coexist hot region schedules.
	HotRegionScheduleLimit uint64 `toml:"hot-region-schedule-limit" json:"hot-region-schedule-limit"`
	// HotRegionCacheHitThreshold is the cache hits threshold of the hot region.
	// If the number of times a region hits the hot cache is greater than this
	// threshold, it is considered a hot region.
	HotRegionCacheHitsThreshold uint64 `toml:"hot-region-cache-hits-threshold" json:"hot-region-cache-hits-threshold"`
	// StoreBalanceRate is the maximum of balance rate for each store.
	// WARN: StoreBalanceRate is deprecated.
	StoreBalanceRate float64 `toml:"store-balance-rate" json:"store-balance-rate,omitempty"`
	// StoreLimit is the limit of scheduling for stores.
	StoreLimit map[uint64]StoreLimitConfig `toml:"store-limit" json:"store-limit"`
	// TolerantSizeRatio is the ratio of buffer size for balance scheduler.
	TolerantSizeRatio float64 `toml:"tolerant-size-ratio" json:"tolerant-size-ratio"`
	//
	//      high space stage         transition stage           low space stage
	//   |--------------------|-----------------------------|-------------------------|
	//   ^                    ^                             ^                         ^
	//   0       HighSpaceRatio * capacity       LowSpaceRatio * capacity          capacity
	//
	// LowSpaceRatio is the lowest usage ratio of store which regraded as low space.
	// When in low space, store region score increases to very large and varies inversely with available size.
	LowSpaceRatio float64 `toml:"low-space-ratio" json:"low-space-ratio"`
	// HighSpaceRatio is the highest usage ratio of store which regraded as high space.
	// High space means there is a lot of spare capacity, and store region score varies directly with used size.
	HighSpaceRatio float64 `toml:"high-space-ratio" json:"high-space-ratio"`
	// RegionScoreFormulaVersion is used to control the formula used to calculate region score.
	RegionScoreFormulaVersion string `toml:"region-score-formula-version" json:"region-score-formula-version"`
	// SchedulerMaxWaitingOperator is the max coexist operators for each scheduler.
	SchedulerMaxWaitingOperator uint64 `toml:"scheduler-max-waiting-operator" json:"scheduler-max-waiting-operator"`
	// WARN: DisableLearner is deprecated.
	// DisableLearner is the option to disable using AddLearnerNode instead of AddNode.
	DisableLearner bool `toml:"disable-raft-learner" json:"disable-raft-learner,string,omitempty"`
	// DisableRemoveDownReplica is the option to prevent replica checker from
	// removing down replicas.
	// WARN: DisableRemoveDownReplica is deprecated.
	DisableRemoveDownReplica bool `toml:"disable-remove-down-replica" json:"disable-remove-down-replica,string,omitempty"`
	// DisableReplaceOfflineReplica is the option to prevent replica checker from
	// replacing offline replicas.
	// WARN: DisableReplaceOfflineReplica is deprecated.
	DisableReplaceOfflineReplica bool `toml:"disable-replace-offline-replica" json:"disable-replace-offline-replica,string,omitempty"`
	// DisableMakeUpReplica is the option to prevent replica checker from making up
	// replicas when replica count is less than expected.
	// WARN: DisableMakeUpReplica is deprecated.
	DisableMakeUpReplica bool `toml:"disable-make-up-replica" json:"disable-make-up-replica,string,omitempty"`
	// DisableRemoveExtraReplica is the option to prevent replica checker from
	// removing extra replicas.
	// WARN: DisableRemoveExtraReplica is deprecated.
	DisableRemoveExtraReplica bool `toml:"disable-remove-extra-replica" json:"disable-remove-extra-replica,string,omitempty"`
	// DisableLocationReplacement is the option to prevent replica checker from
	// moving replica to a better location.
	// WARN: DisableLocationReplacement is deprecated.
	DisableLocationReplacement bool `toml:"disable-location-replacement" json:"disable-location-replacement,string,omitempty"`

	// EnableRemoveDownReplica is the option to enable replica checker to remove down replica.
	EnableRemoveDownReplica bool `toml:"enable-remove-down-replica" json:"enable-remove-down-replica,string"`
	// EnableReplaceOfflineReplica is the option to enable replica checker to replace offline replica.
	EnableReplaceOfflineReplica bool `toml:"enable-replace-offline-replica" json:"enable-replace-offline-replica,string"`
	// EnableMakeUpReplica is the option to enable replica checker to make up replica.
	EnableMakeUpReplica bool `toml:"enable-make-up-replica" json:"enable-make-up-replica,string"`
	// EnableRemoveExtraReplica is the option to enable replica checker to remove extra replica.
	EnableRemoveExtraReplica bool `toml:"enable-remove-extra-replica" json:"enable-remove-extra-replica,string"`
	// EnableLocationReplacement is the option to enable replica checker to move replica to a better location.
	EnableLocationReplacement bool `toml:"enable-location-replacement" json:"enable-location-replacement,string"`
	// EnableDebugMetrics is the option to enable debug metrics.
	EnableDebugMetrics bool `toml:"enable-debug-metrics" json:"enable-debug-metrics,string"`
	// EnableJointConsensus is the option to enable using joint consensus as a operator step.
	EnableJointConsensus bool `toml:"enable-joint-consensus" json:"enable-joint-consensus,string"`

	// Schedulers support for loading customized schedulers
	Schedulers SchedulerConfigs `toml:"schedulers" json:"schedulers-v2"` // json v2 is for the sake of compatible upgrade

	// Only used to display
	SchedulersPayload map[string]interface{} `toml:"schedulers-payload" json:"schedulers-payload"`

	// StoreLimitMode can be auto or manual, when set to auto,
	// PD tries to change the store limit values according to
	// the load state of the cluster dynamically. User can
	// overwrite the auto-tuned value by pd-ctl, when the value
	// is overwritten, the value is fixed until it is deleted.
	// Default: manual
	StoreLimitMode string `toml:"store-limit-mode" json:"store-limit-mode"`
}

// SchedulerConfig is customized scheduler configuration
type SchedulerConfig struct {
	Type        string   `toml:"type" json:"type"`
	Args        []string `toml:"args" json:"args"`
	Disable     bool     `toml:"disable" json:"disable"`
	ArgsPayload string   `toml:"args-payload" json:"args-payload"`
}

// ReplicationConfig is the replication configuration.
type ReplicationConfig struct {
	// MaxReplicas is the number of replicas for each region.
	MaxReplicas uint64 `toml:"max-replicas" json:"max-replicas"`

	// The label keys specified the location of a store.
	// The placement priorities is implied by the order of label keys.
	// For example, ["zone", "rack"] means that we should place replicas to
	// different zones first, then to different racks if we don't have enough zones.
	LocationLabels typeutil.StringSlice `toml:"location-labels" json:"location-labels"`
	// StrictlyMatchLabel strictly checks if the label of TiKV is matched with LocationLabels.
	StrictlyMatchLabel bool `toml:"strictly-match-label" json:"strictly-match-label,string"`

	// When PlacementRules feature is enabled. MaxReplicas, LocationLabels and IsolationLabels are not used any more.
	EnablePlacementRules bool `toml:"enable-placement-rules" json:"enable-placement-rules,string"`

	// IsolationLevel is used to isolate replicas explicitly and forcibly if it's not empty.
	// Its value must be empty or one of LocationLabels.
	// Example:
	// location-labels = ["zone", "rack", "host"]
	// isolation-level = "zone"
	// With configuration like above, PD ensure that all replicas be placed in different zones.
	// Even if a zone is down, PD will not try to make up replicas in other zone
	// because other zones already have replicas on it.
	IsolationLevel string `toml:"isolation-level" json:"isolation-level"`
}

// PDServerConfig is the configuration for pd server.
type PDServerConfig struct {
	// UseRegionStorage enables the independent region storage.
	UseRegionStorage bool `toml:"use-region-storage" json:"use-region-storage,string"`
	// MaxResetTSGap is the max gap to reset the TSO.
	MaxResetTSGap typeutil.Duration `toml:"max-gap-reset-ts" json:"max-gap-reset-ts"`
	// KeyType is option to specify the type of keys.
	// There are some types supported: ["table", "raw", "txn"], default: "table"
	KeyType string `toml:"key-type" json:"key-type"`
	// RuntimeServices is the running the running extension services.
	RuntimeServices typeutil.StringSlice `toml:"runtime-services" json:"runtime-services"`
	// MetricStorage is the cluster metric storage.
	// Currently we use prometheus as metric storage, we may use PD/TiKV as metric storage later.
	MetricStorage string `toml:"metric-storage" json:"metric-storage"`
	// There are some values supported: "auto", "none", or a specific address, default: "auto"
	DashboardAddress string `toml:"dashboard-address" json:"dashboard-address"`
	// TraceRegionFlow the option to update flow information of regions.
	// WARN: TraceRegionFlow is deprecated.
	TraceRegionFlow bool `toml:"trace-region-flow" json:"trace-region-flow,string,omitempty"`
	// FlowRoundByDigit used to discretization processing flow information.
	FlowRoundByDigit int `toml:"flow-round-by-digit" json:"flow-round-by-digit"`
}

type PdConfig struct {
	ClientUrls          string `toml:"client-urls" json:"client-urls"`
	PeerUrls            string `toml:"peer-urls" json:"peer-urls"`
	AdvertiseClientUrls string `toml:"advertise-client-urls" json:"advertise-client-urls"`
	AdvertisePeerUrls   string `toml:"advertise-peer-urls" json:"advertise-peer-urls"`
	Name                string `toml:"name" json:"name"`
	DataDir             string `toml:"data-dir" json:"data-dir"`
	ForceNewCluster     bool   `toml:"force-new-cluster" json:"force-new-cluster"`
	EnableGrpcGateway   bool   `toml:"enable-grpc-gateway" json:"enable-grpc-gateway"`
	InitialCluster      string `toml:"initial-cluster" json:"initial-cluster"`
	InitialClusterState string `toml:"initial-cluster-state" json:"initial-cluster-state"`
	InitialClusterToken string `toml:"initial-cluster-token" json:"initial-cluster-token"`
	Join                string `toml:"join" json:"join"`
	Lease               int    `toml:"lease" json:"lease"`
	Log                 struct {
		Level            string `toml:"level" json:"level"`
		Format           string `toml:"format" json:"format"`
		DisableTimestamp bool   `toml:"disable-timestamp" json:"disable-timestamp"`
		File             struct {
			Filename   string `toml:"filename" json:"filename"`
			MaxSize    int    `toml:"max-size" json:"max-size"`
			MaxDays    int    `toml:"max-days" json:"max-days"`
			MaxBackups int    `toml:"max-backups" json:"max-backups"`
		} `toml:"file" json:"file"`
		Development         bool                `toml:"development" json:"development"`
		DisableCaller       bool                `toml:"disable-caller" json:"disable-caller"`
		DisableStacktrace   bool                `toml:"disable-stacktrace" json:"disable-stacktrace"`
		DisableErrorVerbose bool                `toml:"disable-error-verbose" json:"disable-error-verbose"`
		Sampling            *zap.SamplingConfig `toml:"sampling" json:"sampling"`
	} `toml:"log" json:"log"`
	TsoSaveInterval           string `toml:"tso-save-interval" json:"tso-save-interval"`
	TsoUpdatePhysicalInterval string `toml:"tso-update-physical-interval" json:"tso-update-physical-interval"`
	EnableLocalTso            bool   `toml:"enable-local-tso" json:"enable-local-tso"`
	Metric                    struct {
		Job      string `toml:"job" json:"job"`
		Address  string `toml:"address" json:"address"`
		Interval string `toml:"interval" json:"interval"`
	} `toml:"metric" json:"metric"`
	Schedule       ScheduleConfig    `toml:"schedule" json:"schedule"`
	Replication    ReplicationConfig `toml:"replication" json:"replication"`
	PdServer       PDServerConfig    `toml:"pd-server" json:"pd-server"`
	ClusterVersion string            `toml:"cluster-version" json:"cluster-version"`
	Labels         struct {
	} `toml:"labels" json:"labels"`
	QuotaBackendBytes         string `toml:"quota-backend-bytes" json:"quota-backend-bytes"`
	AutoCompactionMode        string `toml:"auto-compaction-mode" json:"auto-compaction-mode"`
	AutoCompactionRetentionV2 string `toml:"auto-compaction-retention-v2" json:"auto-compaction-retention-v2"`
	TickInterval              string `toml:"TickInterval" json:"TickInterval"`
	ElectionInterval          string `toml:"ElectionInterval" json:"ElectionInterval"`
	PreVote                   bool   `toml:"PreVote" json:"PreVote"`
	Security                  struct {
		CacertPath    string   `toml:"cacert-path" json:"cacert-path"`
		CertPath      string   `toml:"cert-path" json:"cert-path"`
		KeyPath       string   `toml:"key-path" json:"key-path"`
		CertAllowedCn []string `toml:"cert-allowed-cn" json:"cert-allowed-cn"`
		RedactInfoLog bool     `toml:"redact-info-log" json:"redact-info-log"`
		Encryption    struct {
			DataEncryptionMethod  string `toml:"data-encryption-method" json:"data-encryption-method"`
			DataKeyRotationPeriod string `toml:"data-key-rotation-period" json:"data-key-rotation-period"`
			MasterKey             struct {
				Type     string `toml:"type" json:"type"`
				KeyID    string `toml:"key-id" json:"key-id"`
				Region   string `toml:"region" json:"region"`
				Endpoint string `toml:"endpoint" json:"endpoint"`
				Path     string `toml:"path" json:"path"`
			} `toml:"master-key" json:"master-key"`
		} `toml:"encryption" json:"encryption"`
	} `toml:"security" json:"security"`
	LabelProperty struct {
	} `toml:"label-property" json:"label-property"`
	WarningMsgs                 []string `toml:"WarningMsgs" json:"WarningMsgs"`
	DisableStrictReconfigCheck  bool     `toml:"DisableStrictReconfigCheck" json:"DisableStrictReconfigCheck"`
	HeartbeatStreamBindInterval string   `toml:"HeartbeatStreamBindInterval" json:"HeartbeatStreamBindInterval"`
	LeaderPriorityCheckInterval string   `toml:"LeaderPriorityCheckInterval" json:"LeaderPriorityCheckInterval"`
	Dashboard                   struct {
		TidbCacertPath     string `toml:"tidb-cacert-path" json:"tidb-cacert-path"`
		TidbCertPath       string `toml:"tidb-cert-path" json:"tidb-cert-path"`
		TidbKeyPath        string `toml:"tidb-key-path" json:"tidb-key-path"`
		PublicPathPrefix   string `toml:"public-path-prefix" json:"public-path-prefix"`
		InternalProxy      bool   `toml:"internal-proxy" json:"internal-proxy"`
		EnableTelemetry    bool   `toml:"enable-telemetry" json:"enable-telemetry"`
		EnableExperimental bool   `toml:"enable-experimental" json:"enable-experimental"`
	} `toml:"dashboard" json:"dashboard"`
	ReplicationMode struct {
		ReplicationMode string `toml:"replication-mode" json:"replication-mode"`
		DrAutoSync      struct {
			LabelKey         string `toml:"label-key" json:"label-key"`
			Primary          string `toml:"primary" json:"primary"`
			Dr               string `toml:"dr" json:"dr"`
			PrimaryReplicas  int    `toml:"primary-replicas" json:"primary-replicas"`
			DrReplicas       int    `toml:"dr-replicas" json:"dr-replicas"`
			WaitStoreTimeout string `toml:"wait-store-timeout" json:"wait-store-timeout"`
			WaitSyncTimeout  string `toml:"wait-sync-timeout" json:"wait-sync-timeout"`
			WaitAsyncTimeout string `toml:"wait-async-timeout" json:"wait-async-timeout"`
		} `toml:"dr-auto-sync" json:"dr-auto-sync"`
	} `toml:"replication-mode" json:"replication-mode"`
}

type PdConfigData struct {
	*PdConfig
	Port int // TODO move to meta
	Host string
}

func (cfg *PdConfigData) GetPort() int {
	return cfg.Port
}

func (cfg *PdConfigData) GetHost() string {
	return cfg.Host
}

func (cfg *PdConfigData) GetComponent() string {
	return PdComponentName
}

func (cfg *PdConfigData) ActingName() string {
	return cfg.GetComponent()
}

func (cfg *PdConfigData) CheckNil() bool {
	return cfg.PdConfig == nil
}

func (cfg *PdConfigData) GetValueByTagPath(tagPath string) reflect.Value {
	tags := strings.Split(tagPath, ".")
	if len(tags) <= 1 {
		return reflect.ValueOf(cfg)
	}
	value := utils.VisitByTagPath(reflect.ValueOf(cfg.PdConfig), tags, 0)
	return value
}

func NewPdConfigData() *PdConfigData {
	return &PdConfigData{PdConfig: &PdConfig{}}
}
