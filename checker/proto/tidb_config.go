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
	"encoding/json"
	"reflect"
	"strings"

	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/errors"
)

// nullableBool defaults unset bool options to unset instead of false, which enables us to know if the user has set 2
// conflict options at the same time.
type nullableBool struct {
	IsValid bool
	IsTrue  bool
}

var (
	nbUnset = nullableBool{false, false}
	nbFalse = nullableBool{true, false}
	nbTrue  = nullableBool{true, true}
)

func (b *nullableBool) toBool() bool {
	return b.IsValid && b.IsTrue
}

func (b nullableBool) MarshalJSON() ([]byte, error) {
	switch b {
	case nbTrue:
		return json.Marshal(true)
	case nbFalse:
		return json.Marshal(false)
	default:
		return json.Marshal(nil)
	}
}

func (b *nullableBool) UnmarshalText(text []byte) error {
	str := string(text)
	switch str {
	case "", "null":
		*b = nbUnset
		return nil
	case "true":
		*b = nbTrue
	case "false":
		*b = nbFalse
	default:
		*b = nbUnset
		return errors.New("Invalid value for bool type: " + str)
	}
	return nil
}

func (b nullableBool) MarshalText() ([]byte, error) {
	if !b.IsValid {
		return []byte(""), nil
	}
	if b.IsTrue {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}

func (b *nullableBool) UnmarshalJSON(data []byte) error {
	var err error
	var v interface{}
	if err = json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch raw := v.(type) {
	case bool:
		*b = nullableBool{true, raw}
	default:
		*b = nbUnset
	}
	return err
}

type TidbLogConfig struct {
	Level             string       `json:"level"`
	Format            string       `json:"format"`
	DisableTimestamp  nullableBool `json:"disable-timestamp"`
	EnableTimestamp   nullableBool `json:"enable-timestamp"`
	DisableErrorStack nullableBool `json:"disable-error-stack"`
	EnableErrorStack  nullableBool `json:"enable-error-stack"`
	File              struct {
		Filename   string `json:"filename"`
		MaxSize    int    `json:"max-size"`
		MaxDays    int    `json:"max-days"`
		MaxBackups int    `json:"max-backups"`
	} `json:"file"`
	EnableSlowLog       bool   `json:"enable-slow-log"`
	SlowQueryFile       string `json:"slow-query-file"`
	SlowThreshold       int    `json:"slow-threshold"`
	ExpensiveThreshold  int    `json:"expensive-threshold"`
	QueryLogMaxLen      int    `json:"query-log-max-len"`
	RecordPlanInSlowLog int    `json:"record-plan-in-slow-log"`
}

type TidbConfig struct {
	Host                string        `json:"host"`
	AdvertiseAddress    string        `json:"advertise-address"`
	Port                int           `json:"port"`
	Cors                string        `json:"cors"`
	Store               string        `json:"store"`
	Path                string        `json:"path"`
	Socket              string        `json:"socket"`
	Lease               string        `json:"lease"`
	RunDdl              bool          `json:"run-ddl"`
	SplitTable          bool          `json:"split-table"`
	TokenLimit          int           `json:"token-limit"`
	OomUseTmpStorage    bool          `json:"oom-use-tmp-storage"`
	TmpStoragePath      string        `json:"tmp-storage-path"`
	OomAction           string        `json:"oom-action"`
	MemQuotaQuery       int           `json:"mem-quota-query"`
	TmpStorageQuota     int           `json:"tmp-storage-quota"`
	EnableStreaming     bool          `json:"enable-streaming"`
	EnableBatchDml      bool          `json:"enable-batch-dml"`
	LowerCaseTableNames int           `json:"lower-case-table-names"`
	ServerVersion       string        `json:"server-version"`
	Log                 TidbLogConfig `json:"log"`
	Security            struct {
		SkipGrantTable         bool     `toml:"skip-grant-table" json:"skip-grant-table"`
		SSLCA                  string   `toml:"ssl-ca" json:"ssl-ca"`
		SSLCert                string   `toml:"ssl-cert" json:"ssl-cert"`
		SSLKey                 string   `toml:"ssl-key" json:"ssl-key"`
		RequireSecureTransport bool     `toml:"require-secure-transport" json:"require-secure-transport"`
		ClusterSSLCA           string   `toml:"cluster-ssl-ca" json:"cluster-ssl-ca"`
		ClusterSSLCert         string   `toml:"cluster-ssl-cert" json:"cluster-ssl-cert"`
		ClusterSSLKey          string   `toml:"cluster-ssl-key" json:"cluster-ssl-key"`
		ClusterVerifyCN        []string `toml:"cluster-verify-cn" json:"cluster-verify-cn"`
		// If set to "plaintext", the spilled files will not be encrypted.
		SpilledFileEncryptionMethod string `toml:"spilled-file-encryption-method" json:"spilled-file-encryption-method"`
		// EnableSEM prevents SUPER users from having full access.
		EnableSEM bool `toml:"enable-sem" json:"enable-sem"`
		// Allow automatic TLS certificate generation
		AutoTLS         bool   `toml:"auto-tls" json:"auto-tls"`
		MinTLSVersion   string `toml:"tls-version" json:"tls-version"`
		RSAKeySize      int    `toml:"rsa-key-size" json:"rsa-key-size"`
		SecureBootstrap bool   `toml:"secure-bootstrap" json:"secure-bootstrap"`
	} `json:"security"`
	Status struct {
		StatusHost      string `json:"status-host"`
		MetricsAddr     string `json:"metrics-addr"`
		StatusPort      int    `json:"status-port"`
		MetricsInterval int    `json:"metrics-interval"`
		ReportStatus    bool   `json:"report-status"`
		RecordDbQPS     bool   `json:"record-db-qps"`
	} `json:"status"`
	Performance struct {
		MaxProcs uint `toml:"max-procs" json:"max-procs"`
		// Deprecated: use ServerMemoryQuota instead
		MaxMemory             uint64  `toml:"max-memory" json:"max-memory"`
		ServerMemoryQuota     uint64  `toml:"server-memory-quota" json:"server-memory-quota"`
		MemoryUsageAlarmRatio float64 `toml:"memory-usage-alarm-ratio" json:"memory-usage-alarm-ratio"`
		StatsLease            string  `toml:"stats-lease" json:"stats-lease"`
		StmtCountLimit        uint    `toml:"stmt-count-limit" json:"stmt-count-limit"`
		FeedbackProbability   float64 `toml:"feedback-probability" json:"feedback-probability"`
		QueryFeedbackLimit    uint    `toml:"query-feedback-limit" json:"query-feedback-limit"`
		PseudoEstimateRatio   float64 `toml:"pseudo-estimate-ratio" json:"pseudo-estimate-ratio"`
		ForcePriority         string  `toml:"force-priority" json:"force-priority"`
		BindInfoLease         string  `toml:"bind-info-lease" json:"bind-info-lease"`
		TxnEntrySizeLimit     uint64  `toml:"txn-entry-size-limit" json:"txn-entry-size-limit"`
		TxnTotalSizeLimit     uint64  `toml:"txn-total-size-limit" json:"txn-total-size-limit"`
		TCPKeepAlive          bool    `toml:"tcp-keep-alive" json:"tcp-keep-alive"`
		TCPNoDelay            bool    `toml:"tcp-no-delay" json:"tcp-no-delay"`
		CrossJoin             bool    `toml:"cross-join" json:"cross-join"`
		RunAutoAnalyze        bool    `toml:"run-auto-analyze" json:"run-auto-analyze"`
		DistinctAggPushDown   bool    `toml:"distinct-agg-push-down" json:"distinct-agg-push-down"`
		CommitterConcurrency  int     `toml:"committer-concurrency" json:"committer-concurrency"`
		MaxTxnTTL             uint64  `toml:"max-txn-ttl" json:"max-txn-ttl"`
		MemProfileInterval    string  `toml:"mem-profile-interval" json:"mem-profile-interval"`
		IndexUsageSyncLease   string  `toml:"index-usage-sync-lease" json:"index-usage-sync-lease"`
		PlanReplayerGCLease   string  `toml:"plan-replayer-gc-lease" json:"plan-replayer-gc-lease"`
		GOGC                  int     `toml:"gogc" json:"gogc"`
		EnforceMPP            bool    `toml:"enforce-mpp" json:"enforce-mpp"`
	} `json:"performance"`
	PreparedPlanCache struct {
		Enabled          bool    `json:"enabled"`
		Capacity         int     `json:"capacity"`
		MemoryGuardRatio float64 `json:"memory-guard-ratio"`
	} `json:"prepared-plan-cache"`
	Opentracing struct {
		Enable     bool `json:"enable"`
		RPCMetrics bool `json:"rpc-metrics"`
		Sampler    struct {
			Type                    string `json:"type"`
			Param                   int    `json:"param"`
			SamplingServerURL       string `json:"sampling-server-url"`
			MaxOperations           int    `json:"max-operations"`
			SamplingRefreshInterval int    `json:"sampling-refresh-interval"`
		} `json:"sampler"`
		Reporter struct {
			QueueSize           int    `json:"queue-size"`
			BufferFlushInterval int    `json:"buffer-flush-interval"`
			LogSpans            bool   `json:"log-spans"`
			LocalAgentHostPort  string `json:"local-agent-host-port"`
		} `json:"reporter"`
	} `json:"opentracing"`
	ProxyProtocol struct {
		Networks      string `json:"networks"`
		HeaderTimeout int    `json:"header-timeout"`
	} `json:"proxy-protocol"`
	PdClient struct {
		PdServerTimeout int `json:"pd-server-timeout"`
	} `json:"pd-client"`
	TikvClient struct {
		GrpcConnectionCount  int    `json:"grpc-connection-count"`
		GrpcKeepaliveTime    int    `json:"grpc-keepalive-time"`
		GrpcKeepaliveTimeout int    `json:"grpc-keepalive-timeout"`
		GrpcCompressionType  string `json:"grpc-compression-type"`
		CommitTimeout        string `json:"commit-timeout"`
		AsyncCommit          struct {
			KeysLimit         int `json:"keys-limit"`
			TotalKeySizeLimit int `json:"total-key-size-limit"`
			SafeWindow        int `json:"safe-window"`
			AllowedClockDrift int `json:"allowed-clock-drift"`
		} `json:"async-commit"`
		MaxBatchSize         int    `json:"max-batch-size"`
		OverloadThreshold    int    `json:"overload-threshold"`
		MaxBatchWaitTime     int    `json:"max-batch-wait-time"`
		BatchWaitSize        int    `json:"batch-wait-size"`
		EnableChunkRPC       bool   `json:"enable-chunk-rpc"`
		RegionCacheTTL       int    `json:"region-cache-ttl"`
		StoreLimit           int    `json:"store-limit"`
		StoreLivenessTimeout string `json:"store-liveness-timeout"`
		CoprCache            struct {
			CapacityMb int `json:"capacity-mb"`
		} `json:"copr-cache"`
		TTLRefreshedTxnSize int `json:"ttl-refreshed-txn-size"`
	} `json:"tikv-client"`
	Binlog struct {
		Enable       bool   `json:"enable"`
		IgnoreError  bool   `json:"ignore-error"`
		WriteTimeout string `json:"write-timeout"`
		BinlogSocket string `json:"binlog-socket"`
		Strategy     string `json:"strategy"`
	} `json:"binlog"`
	CompatibleKillQuery bool `json:"compatible-kill-query"`
	Plugin              struct {
		Dir  string `json:"dir"`
		Load string `json:"load"`
	} `json:"plugin"`
	PessimisticTxn struct {
		// The max count of retry for a single statement in a pessimistic transaction.
		MaxRetryCount uint `toml:"max-retry-count" json:"max-retry-count"`
		// The max count of deadlock events that will be recorded in the information_schema.deadlocks table.
		DeadlockHistoryCapacity uint `toml:"deadlock-history-capacity" json:"deadlock-history-capacity"`
		// Whether retryable deadlocks (in-statement deadlocks) are collected to the information_schema.deadlocks table.
		DeadlockHistoryCollectRetryable bool `toml:"deadlock-history-collect-retryable" json:"deadlock-history-collect-retryable"`
	} `toml:"pessimistic-txn" json:"pessimistic-txn"`
	CheckMb4ValueInUtf8          bool `json:"check-mb4-value-in-utf8"`
	MaxIndexLength               int  `json:"max-index-length"`
	IndexLimit                   int  `json:"index-limit"`
	TableColumnCountLimit        int  `json:"table-column-count-limit"`
	GracefulWaitBeforeShutdown   int  `json:"graceful-wait-before-shutdown"`
	AlterPrimaryKey              bool `json:"alter-primary-key"`
	TreatOldVersionUtf8AsUtf8Mb4 bool `json:"treat-old-version-utf8-as-utf8mb4"`
	EnableTableLock              bool `json:"enable-table-lock"`
	DelayCleanTableLock          int  `json:"delay-clean-table-lock"`
	SplitRegionMaxNum            int  `json:"split-region-max-num"`
	StmtSummary                  struct {
		Enable              bool `json:"enable"`
		EnableInternalQuery bool `json:"enable-internal-query"`
		MaxStmtCount        int  `json:"max-stmt-count"`
		MaxSQLLength        int  `json:"max-sql-length"`
		RefreshInterval     int  `json:"refresh-interval"`
		HistorySize         int  `json:"history-size"`
	} `json:"stmt-summary"`
	RepairMode      bool          `json:"repair-mode"`
	RepairTableList []interface{} `json:"repair-table-list"`
	IsolationRead   struct {
		Engines []string `json:"engines"`
	} `json:"isolation-read"`
	MaxServerConnections                 int  `json:"max-server-connections"`
	NewCollationsEnabledOnFirstBootstrap bool `json:"new_collations_enabled_on_first_bootstrap"`
	Experimental                         struct {
		AllowExpressionIndex bool `json:"allow-expression-index"`
	} `json:"experimental"`
	EnableCollectExecutionInfo bool `json:"enable-collect-execution-info"`
	SkipRegisterToDashboard    bool `json:"skip-register-to-dashboard"`
	EnableTelemetry            bool `json:"enable-telemetry"`
	Labels                     struct {
	} `json:"labels"`
	EnableGlobalIndex             bool `json:"enable-global-index"`
	DeprecateIntegerDisplayLength bool `json:"deprecate-integer-display-length"`
	EnableEnumLengthLimit         bool `json:"enable-enum-length-limit"`
	StoresRefreshInterval         int  `json:"stores-refresh-interval"`
	EnableTCP4Only                bool `json:"enable-tcp4-only"`
	EnableForwarding              bool `json:"enable-forwarding"`
}

type TidbConfigData struct {
	*TidbConfig     // embedded field
	Port        int // TODO move to meta
	Host        string
}

func (cfg *TidbConfigData) GetComponent() string {
	return TidbComponentName
}

func (cfg *TidbConfigData) GetValueByTagPath(tagPath string) reflect.Value {
	tags := strings.Split(tagPath, ".")
	if len(tags) == 0 {
		return reflect.ValueOf(cfg.TidbConfig)
	}
	value := utils.VisitByTagPath(reflect.ValueOf(cfg.TidbConfig), tags, 0)
	return value
}

func (cfg *TidbConfigData) GetPort() int {
	return cfg.Port
}

func (cfg *TidbConfigData) GetHost() string {
	return cfg.Host
}

func (cfg *TidbConfigData) CheckNil() bool {
	return cfg.TidbConfig == nil
}

func (cfg *TidbConfigData) ActingName() string {
	return cfg.GetComponent()
}

func NewTidbConfigData() *TidbConfigData {
	return &TidbConfigData{TidbConfig: &TidbConfig{
		Log: TidbLogConfig{
			Level:  "info",
			Format: "text",
			File: struct {
				Filename   string `json:"filename"`
				MaxSize    int    `json:"max-size"`
				MaxDays    int    `json:"max-days"`
				MaxBackups int    `json:"max-backups"`
			}{
				MaxSize: 300,
			},
			EnableSlowLog:       true,
			SlowQueryFile:       "tidb-slow.log",
			SlowThreshold:       300,
			ExpensiveThreshold:  10000,
			QueryLogMaxLen:      4096,
			RecordPlanInSlowLog: 1,
		},
	}}
}
