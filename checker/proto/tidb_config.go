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
	"time"

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

// Log is the log section of config.
type Log struct {
	// Log level.
	Level string `toml:"level" json:"level"`
	// Log format, one of json or text.
	Format string `toml:"format" json:"format"`
	// Disable automatic timestamps in output. Deprecated: use EnableTimestamp instead.
	DisableTimestamp nullableBool `toml:"disable-timestamp" json:"disable-timestamp"`
	// EnableTimestamp enables automatic timestamps in log output.
	EnableTimestamp nullableBool `toml:"enable-timestamp" json:"enable-timestamp"`
	// DisableErrorStack stops annotating logs with the full stack error
	// message. Deprecated: use EnableErrorStack instead.
	DisableErrorStack nullableBool `toml:"disable-error-stack" json:"disable-error-stack"`
	// EnableErrorStack enables annotating logs with the full stack error
	// message.
	EnableErrorStack nullableBool `toml:"enable-error-stack" json:"enable-error-stack"`
	// File log config.
	File struct {
		Filename   string `json:"filename"`
		MaxSize    int    `json:"max-size"`
		MaxDays    int    `json:"max-days"`
		MaxBackups int    `json:"max-backups"`
	} `toml:"file" json:"file"`

	EnableSlowLog       bool   `toml:"enable-slow-log" json:"enable-slow-log"`
	SlowQueryFile       string `toml:"slow-query-file" json:"slow-query-file"`
	SlowThreshold       uint64 `toml:"slow-threshold" json:"slow-threshold"`
	ExpensiveThreshold  uint   `toml:"expensive-threshold" json:"expensive-threshold"`
	QueryLogMaxLen      uint64 `toml:"query-log-max-len" json:"query-log-max-len"`
	RecordPlanInSlowLog uint32 `toml:"record-plan-in-slow-log" json:"record-plan-in-slow-log"`
}

// OpenTracing is the opentracing section of the config.
type OpenTracing struct {
	Enable     bool                `toml:"enable" json:"enable"`
	RPCMetrics bool                `toml:"rpc-metrics" json:"rpc-metrics"`
	Sampler    OpenTracingSampler  `toml:"sampler" json:"sampler"`
	Reporter   OpenTracingReporter `toml:"reporter" json:"reporter"`
}

// OpenTracingSampler is the config for opentracing sampler.
// See https://godoc.org/github.com/uber/jaeger-client-go/config#SamplerConfig
type OpenTracingSampler struct {
	Type                    string        `toml:"type" json:"type"`
	Param                   float64       `toml:"param" json:"param"`
	SamplingServerURL       string        `toml:"sampling-server-url" json:"sampling-server-url"`
	MaxOperations           int           `toml:"max-operations" json:"max-operations"`
	SamplingRefreshInterval time.Duration `toml:"sampling-refresh-interval" json:"sampling-refresh-interval"`
}

// OpenTracingReporter is the config for opentracing reporter.
// See https://godoc.org/github.com/uber/jaeger-client-go/config#ReporterConfig
type OpenTracingReporter struct {
	QueueSize           int           `toml:"queue-size" json:"queue-size"`
	BufferFlushInterval time.Duration `toml:"buffer-flush-interval" json:"buffer-flush-interval"`
	LogSpans            bool          `toml:"log-spans" json:"log-spans"`
	LocalAgentHostPort  string        `toml:"local-agent-host-port" json:"local-agent-host-port"`
}

// TiKVClient is the config for tikv client.
type TiKVClient struct {
	// GrpcConnectionCount is the max gRPC connections that will be established
	// with each tikv-server.
	GrpcConnectionCount uint `toml:"grpc-connection-count" json:"grpc-connection-count"`
	// After a duration of this time in seconds if the client doesn't see any activity it pings
	// the server to see if the transport is still alive.
	GrpcKeepAliveTime uint `toml:"grpc-keepalive-time" json:"grpc-keepalive-time"`
	// After having pinged for keepalive check, the client waits for a duration of Timeout in seconds
	// and if no activity is seen even after that the connection is closed.
	GrpcKeepAliveTimeout uint `toml:"grpc-keepalive-timeout" json:"grpc-keepalive-timeout"`
	// GrpcCompressionType is the compression type for gRPC channel: none or gzip.
	GrpcCompressionType string `toml:"grpc-compression-type" json:"grpc-compression-type"`
	// CommitTimeout is the max time which command 'commit' will wait.
	CommitTimeout string      `toml:"commit-timeout" json:"commit-timeout"`
	AsyncCommit   AsyncCommit `toml:"async-commit" json:"async-commit"`
	// MaxBatchSize is the max batch size when calling batch commands API.
	MaxBatchSize uint `toml:"max-batch-size" json:"max-batch-size"`
	// If TiKV load is greater than this, TiDB will wait for a while to avoid little batch.
	OverloadThreshold uint `toml:"overload-threshold" json:"overload-threshold"`
	// MaxBatchWaitTime in nanosecond is the max wait time for batch.
	MaxBatchWaitTime time.Duration `toml:"max-batch-wait-time" json:"max-batch-wait-time"`
	// BatchWaitSize is the max wait size for batch.
	BatchWaitSize uint `toml:"batch-wait-size" json:"batch-wait-size"`
	// EnableChunkRPC indicate the data encode in chunk format for coprocessor requests.
	EnableChunkRPC bool `toml:"enable-chunk-rpc" json:"enable-chunk-rpc"`
	// If a Region has not been accessed for more than the given duration (in seconds), it
	// will be reloaded from the PD.
	RegionCacheTTL uint `toml:"region-cache-ttl" json:"region-cache-ttl"`
	// If a store has been up to the limit, it will return error for successive request to
	// prevent the store occupying too much token in dispatching level.
	StoreLimit int64 `toml:"store-limit" json:"store-limit"`
	// StoreLivenessTimeout is the timeout for store liveness check request.
	StoreLivenessTimeout string           `toml:"store-liveness-timeout" json:"store-liveness-timeout"`
	CoprCache            CoprocessorCache `toml:"copr-cache" json:"copr-cache"`
	// TTLRefreshedTxnSize controls whether a transaction should update its TTL or not.
	TTLRefreshedTxnSize      int64  `toml:"ttl-refreshed-txn-size" json:"ttl-refreshed-txn-size"`
	ResolveLockLiteThreshold uint64 `toml:"resolve-lock-lite-threshold" json:"resolve-lock-lite-threshold"`
}

// AsyncCommit is the config for the async commit feature. The switch to enable it is a system variable.
type AsyncCommit struct {
	// Use async commit only if the number of keys does not exceed KeysLimit.
	KeysLimit uint `toml:"keys-limit" json:"keys-limit"`
	// Use async commit only if the total size of keys does not exceed TotalKeySizeLimit.
	TotalKeySizeLimit uint64 `toml:"total-key-size-limit" json:"total-key-size-limit"`
	// The duration within which is safe for async commit or 1PC to commit with an old schema.
	// The following two fields should NOT be modified in most cases. If both async commit
	// and 1PC are disabled in the whole cluster, they can be set to zero to avoid waiting in DDLs.
	SafeWindow time.Duration `toml:"safe-window" json:"safe-window"`
	// The duration in addition to SafeWindow to make DDL safe.
	AllowedClockDrift time.Duration `toml:"allowed-clock-drift" json:"allowed-clock-drift"`
}

// CoprocessorCache is the config for coprocessor cache.
type CoprocessorCache struct {
	// The capacity in MB of the cache. Zero means disable coprocessor cache.
	CapacityMB float64 `toml:"capacity-mb" json:"capacity-mb"`

	// No json fields for below config. Intend to hide them.

	// Only cache requests that containing small number of ranges. May to be changed in future.
	AdmissionMaxRanges uint64 `toml:"admission-max-ranges" json:"-"`
	// Only cache requests whose result set is small.
	AdmissionMaxResultMB float64 `toml:"admission-max-result-mb" json:"-"`
	// Only cache requests takes notable time to process.
	AdmissionMinProcessMs uint64 `toml:"admission-min-process-ms" json:"-"`
}

// Binlog is the config for binlog.
type Binlog struct {
	Enable bool `toml:"enable" json:"enable"`
	// If IgnoreError is true, when writing binlog meets error, TiDB would
	// ignore the error.
	IgnoreError  bool   `toml:"ignore-error" json:"ignore-error"`
	WriteTimeout string `toml:"write-timeout" json:"write-timeout"`
	// Use socket file to write binlog, for compatible with kafka version tidb-binlog.
	BinlogSocket string `toml:"binlog-socket" json:"binlog-socket"`
	// The strategy for sending binlog to pump, value can be "range" or "hash" now.
	Strategy string `toml:"strategy" json:"strategy"`
}

// StmtSummary is the config for statement summary.
type StmtSummary struct {
	// Enable statement summary or not.
	Enable bool `toml:"enable" json:"enable"`
	// Enable summary internal query.
	EnableInternalQuery bool `toml:"enable-internal-query" json:"enable-internal-query"`
	// The maximum number of statements kept in memory.
	MaxStmtCount uint `toml:"max-stmt-count" json:"max-stmt-count"`
	// The maximum length of displayed normalized SQL and sample SQL.
	MaxSQLLength uint `toml:"max-sql-length" json:"max-sql-length"`
	// The refresh interval of statement summary.
	RefreshInterval int `toml:"refresh-interval" json:"refresh-interval"`
	// The maximum history size of statement summary.
	HistorySize int `toml:"history-size" json:"history-size"`
}

// Security is the security section of the config.
type Security struct {
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
}

// Status is the status section of the config.
type Status struct {
	StatusHost      string `toml:"status-host" json:"status-host"`
	MetricsAddr     string `toml:"metrics-addr" json:"metrics-addr"`
	StatusPort      uint   `toml:"status-port" json:"status-port"`
	MetricsInterval uint   `toml:"metrics-interval" json:"metrics-interval"`
	ReportStatus    bool   `toml:"report-status" json:"report-status"`
	RecordQPSbyDB   bool   `toml:"record-db-qps" json:"record-db-qps"`
}

// Performance is the performance section of the config.
type Performance struct {
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
}

type TidbConfig struct {
	Host                        string      `toml:"host" json:"host"`
	AdvertiseAddress            string      `toml:"advertise-address" json:"advertise-address"`
	Port                        uint        `toml:"port" json:"port"`
	Cors                        string      `toml:"cors" json:"cors"`
	Store                       string      `toml:"store" json:"store"`
	Path                        string      `toml:"path" json:"path"`
	Socket                      string      `toml:"socket" json:"socket"`
	Lease                       string      `toml:"lease" json:"lease"`
	RunDDL                      bool        `toml:"run-ddl" json:"run-ddl"`
	SplitTable                  bool        `toml:"split-table" json:"split-table"`
	TokenLimit                  uint        `toml:"token-limit" json:"token-limit"`
	OOMUseTmpStorage            bool        `toml:"oom-use-tmp-storage" json:"oom-use-tmp-storage"`
	TempStoragePath             string      `toml:"tmp-storage-path" json:"tmp-storage-path"`
	OOMAction                   string      `toml:"oom-action" json:"oom-action"`
	MemQuotaQuery               int64       `toml:"mem-quota-query" json:"mem-quota-query"`
	NestedLoopJoinCacheCapacity int64       `toml:"nested-loop-join-cache-capacity" json:"nested-loop-join-cache-capacity"`
	TempStorageQuota            int64       `toml:"tmp-storage-quota" json:"tmp-storage-quota"` // Bytes
	EnableStreaming             bool        `toml:"enable-streaming" json:"enable-streaming"`
	EnableBatchDML              bool        `toml:"enable-batch-dml" json:"enable-batch-dml"`
	LowerCaseTableNames         int         `toml:"lower-case-table-names" json:"lower-case-table-names"`
	ServerVersion               string      `toml:"server-version" json:"server-version"`
	Log                         Log         `toml:"log" json:"log"`
	Security                    Security    `toml:"security" json:"security"`
	Status                      Status      `toml:"status" json:"status"`
	Performance                 Performance `toml:"performance" json:"performance"`
	PreparedPlanCache           struct {
		Enabled          bool    `toml:"enabled" json:"enabled"`
		Capacity         int     `toml:"capacity" json:"capacity"`
		MemoryGuardRatio float64 `toml:"memory-guard-ratio" json:"memory-guard-ratio"`
	} `toml:"prepared-plan-cache" json:"prepared-plan-cache"`
	Opentracing   OpenTracing `toml:"opentracing" json:"opentracing"`
	ProxyProtocol struct {
		Networks      string `toml:"networks" json:"networks"`
		HeaderTimeout int    `toml:"header-timeout" json:"header-timeout"`
	} `toml:"proxy-protocol" json:"proxy-protocol"`
	PdClient struct {
		PdServerTimeout int `toml:"pd-server-timeout" json:"pd-server-timeout"`
	} `toml:"pd-client" json:"pd-client"`
	TikvClient          TiKVClient `toml:"tikv-client" json:"tikv-client"`
	Binlog              Binlog     `toml:"binlog" json:"binlog"`
	CompatibleKillQuery bool       `json:"compatible-kill-query"`
	Plugin              struct {
		Dir  string `toml:"dir" json:"dir"`
		Load string `toml:"load" json:"load"`
	} `toml:"plugin" json:"plugin"`
	PessimisticTxn struct {
		// The max count of retry for a single statement in a pessimistic transaction.
		MaxRetryCount uint `toml:"max-retry-count" json:"max-retry-count"`
		// The max count of deadlock events that will be recorded in the information_schema.deadlocks table.
		DeadlockHistoryCapacity uint `toml:"deadlock-history-capacity" json:"deadlock-history-capacity"`
		// Whether retryable deadlocks (in-statement deadlocks) are collected to the information_schema.deadlocks table.
		DeadlockHistoryCollectRetryable bool `toml:"deadlock-history-collect-retryable" json:"deadlock-history-collect-retryable"`
	} `toml:"pessimistic-txn" json:"pessimistic-txn"`
	CheckMb4ValueInUtf8          bool        `toml:"check-mb4-value-in-utf8" json:"check-mb4-value-in-utf8"`
	MaxIndexLength               int         `toml:"max-index-length" json:"max-index-length"`
	IndexLimit                   int         `toml:"index-limit" json:"index-limit"`
	TableColumnCountLimit        int         `toml:"table-column-count-limit" json:"table-column-count-limit"`
	GracefulWaitBeforeShutdown   int         `toml:"graceful-wait-before-shutdown" json:"graceful-wait-before-shutdown"`
	AlterPrimaryKey              bool        `toml:"alter-primary-key" json:"alter-primary-key"`
	TreatOldVersionUtf8AsUtf8Mb4 bool        `json:"treat-old-version-utf8-as-utf8mb4"`
	EnableTableLock              bool        `toml:"enable-table-lock" json:"enable-table-lock"`
	DelayCleanTableLock          int         `toml:"delay-clean-table-lock" json:"delay-clean-table-lock"`
	SplitRegionMaxNum            int         `toml:"split-region-max-num" json:"split-region-max-num"`
	StmtSummary                  StmtSummary `toml:"stmt-summary" json:"stmt-summary"`
	RepairMode                   bool        `toml:"repair-mode" json:"repair-mode"`
	RepairTableList              []string    `toml:"repair-table-list" json:"repair-table-list"`
	IsolationRead                struct {
		Engines []string `toml:"engines" json:"engines"`
	} `toml:"isolation-read" json:"isolation-read"`
	MaxServerConnections                 int  `toml:"max-server-connections" json:"max-server-connections"`
	NewCollationsEnabledOnFirstBootstrap bool `toml:"new_collations_enabled_on_first_bootstrap" json:"new_collations_enabled_on_first_bootstrap"`
	Experimental                         struct {
		AllowExpressionIndex bool `toml:"allow-expression-index" json:"allow-expression-index"`
	} `toml:"experimental" json:"experimental"`
	EnableCollectExecutionInfo bool `toml:"enable-collect-execution-info" json:"enable-collect-execution-info"`
	SkipRegisterToDashboard    bool `toml:"skip-register-to-dashboard" json:"skip-register-to-dashboard"`
	EnableTelemetry            bool `toml:"enable-telemetry" json:"enable-telemetry"`
	Labels                     struct {
	} `toml:"labels" json:"labels"`
	EnableGlobalIndex             bool `toml:"enable-global-index" json:"enable-global-index"`
	DeprecateIntegerDisplayLength bool `toml:"deprecate-integer-display-length" json:"deprecate-integer-display-length"`
	EnableEnumLengthLimit         bool `toml:"enable-enum-length-limit" json:"enable-enum-length-limit"`
	StoresRefreshInterval         int  `toml:"stores-refresh-interval" json:"stores-refresh-interval"`
	EnableTCP4Only                bool `toml:"enable-tcp4-only" json:"enable-tcp4-only"`
	EnableForwarding              bool `toml:"enable-forwarding" json:"enable-forwarding"`
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
		Log: Log{
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
