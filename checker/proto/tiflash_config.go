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
	"github.com/pingcap/diag/pkg/utils"
	"reflect"
	"strings"
)

type TiflashConfig struct {
	LogLevel             string  `toml:"log-level" json:"log-level"`
	LogFile              string  `toml:"log-file" json:"log-file"`
	LogFormat            string  `toml:"log-format" json:"log-format"`
	SlowLogFile          string  `toml:"slow-log-file" json:"slow-log-file"`
	SlowLogThreshold     string  `toml:"slow-log-threshold" json:"slow-log-threshold"`
	LogRotationTimespan  string  `toml:"log-rotation-timespan" json:"log-rotation-timespan"`
	LogRotationSize      string  `toml:"log-rotation-size" json:"log-rotation-size"`
	EnableIoSnoop        bool    `toml:"enable-io-snoop" json:"enable-io-snoop"`
	AbortOnPanic         bool    `toml:"abort-on-panic" json:"abort-on-panic"`
	MemoryUsageLimit     string  `toml:"memory-usage-limit" json:"memory-usage-limit"`
	MemoryUsageHighWater float64 `toml:"memory-usage-high-water" json:"memory-usage-high-water"`
	Readpool             struct {
		Unified struct {
			MinThreadCount    int    `toml:"min-thread-count" json:"min-thread-count"`
			MaxThreadCount    int    `toml:"max-thread-count" json:"max-thread-count"`
			StackSize         string `toml:"stack-size" json:"stack-size"`
			MaxTasksPerWorker int    `toml:"max-tasks-per-worker" json:"max-tasks-per-worker"`
		} `toml:"unified" json:"unified"`
		Storage struct {
			UseUnifiedPool          bool   `toml:"use-unified-pool" json:"use-unified-pool"`
			HighConcurrency         int    `toml:"high-concurrency" json:"high-concurrency"`
			NormalConcurrency       int    `toml:"normal-concurrency" json:"normal-concurrency"`
			LowConcurrency          int    `toml:"low-concurrency" json:"low-concurrency"`
			MaxTasksPerWorkerHigh   int    `toml:"max-tasks-per-worker-high" json:"max-tasks-per-worker-high"`
			MaxTasksPerWorkerNormal int    `toml:"max-tasks-per-worker-normal" json:"max-tasks-per-worker-normal"`
			MaxTasksPerWorkerLow    int    `toml:"max-tasks-per-worker-low" json:"max-tasks-per-worker-low"`
			StackSize               string `toml:"stack-size" json:"stack-size"`
		} `toml:"storage" json:"storage"`
		Coprocessor struct {
			UseUnifiedPool          bool   `toml:"use-unified-pool" json:"use-unified-pool"`
			HighConcurrency         int    `toml:"high-concurrency" json:"high-concurrency"`
			NormalConcurrency       int    `toml:"normal-concurrency" json:"normal-concurrency"`
			LowConcurrency          int    `toml:"low-concurrency" json:"low-concurrency"`
			MaxTasksPerWorkerHigh   int    `toml:"max-tasks-per-worker-high" json:"max-tasks-per-worker-high"`
			MaxTasksPerWorkerNormal int    `toml:"max-tasks-per-worker-normal" json:"max-tasks-per-worker-normal"`
			MaxTasksPerWorkerLow    int    `toml:"max-tasks-per-worker-low" json:"max-tasks-per-worker-low"`
			StackSize               string `toml:"stack-size" json:"stack-size"`
		} `toml:"coprocessor" json:"coprocessor"`
	} `toml:"readpool" json:"readpool"`
	Server struct {
		Addr                             string `toml:"addr" json:"addr"`
		AdvertiseAddr                    string `toml:"advertise-addr" json:"advertise-addr"`
		EngineAddr                       string `toml:"engine-addr" json:"engine-addr"`
		EngineStoreVersion               string `toml:"engine-store-version" json:"engine-store-version"`
		EngineStoreGitHash               string `toml:"engine-store-git-hash" json:"engine-store-git-hash"`
		StatusAddr                       string `toml:"status-addr" json:"status-addr"`
		AdvertiseStatusAddr              string `toml:"advertise-status-addr" json:"advertise-status-addr"`
		StatusThreadPoolSize             int    `toml:"status-thread-pool-size" json:"status-thread-pool-size"`
		MaxGrpcSendMsgLen                int    `toml:"max-grpc-send-msg-len" json:"max-grpc-send-msg-len"`
		RaftClientGrpcSendMsgBuffer      int    `toml:"raft-client-grpc-send-msg-buffer" json:"raft-client-grpc-send-msg-buffer"`
		RaftClientQueueSize              int    `toml:"raft-client-queue-size" json:"raft-client-queue-size"`
		RaftMsgMaxBatchSize              int    `toml:"raft-msg-max-batch-size" json:"raft-msg-max-batch-size"`
		GrpcCompressionType              string `toml:"grpc-compression-type" json:"grpc-compression-type"`
		GrpcConcurrency                  int    `toml:"grpc-concurrency" json:"grpc-concurrency"`
		GrpcConcurrentStream             int    `toml:"grpc-concurrent-stream" json:"grpc-concurrent-stream"`
		GrpcRaftConnNum                  int    `toml:"grpc-raft-conn-num" json:"grpc-raft-conn-num"`
		GrpcMemoryPoolQuota              int64  `toml:"grpc-memory-pool-quota" json:"grpc-memory-pool-quota"`
		GrpcStreamInitialWindowSize      string `toml:"grpc-stream-initial-window-size" json:"grpc-stream-initial-window-size"`
		GrpcKeepaliveTime                string `toml:"grpc-keepalive-time" json:"grpc-keepalive-time"`
		GrpcKeepaliveTimeout             string `toml:"grpc-keepalive-timeout" json:"grpc-keepalive-timeout"`
		ConcurrentSendSnapLimit          int    `toml:"concurrent-send-snap-limit" json:"concurrent-send-snap-limit"`
		ConcurrentRecvSnapLimit          int    `toml:"concurrent-recv-snap-limit" json:"concurrent-recv-snap-limit"`
		EndPointRecursionLimit           int    `toml:"end-point-recursion-limit" json:"end-point-recursion-limit"`
		EndPointStreamChannelSize        int    `toml:"end-point-stream-channel-size" json:"end-point-stream-channel-size"`
		EndPointBatchRowLimit            int    `toml:"end-point-batch-row-limit" json:"end-point-batch-row-limit"`
		EndPointStreamBatchRowLimit      int    `toml:"end-point-stream-batch-row-limit" json:"end-point-stream-batch-row-limit"`
		EndPointEnableBatchIfPossible    bool   `toml:"end-point-enable-batch-if-possible" json:"end-point-enable-batch-if-possible"`
		EndPointRequestMaxHandleDuration string `toml:"end-point-request-max-handle-duration" json:"end-point-request-max-handle-duration"`
		EndPointMaxConcurrency           int    `toml:"end-point-max-concurrency" json:"end-point-max-concurrency"`
		SnapMaxWriteBytesPerSec          string `toml:"snap-max-write-bytes-per-sec" json:"snap-max-write-bytes-per-sec"`
		SnapMaxTotalSize                 string `toml:"snap-max-total-size" json:"snap-max-total-size"`
		StatsConcurrency                 int    `toml:"stats-concurrency" json:"stats-concurrency"`
		HeavyLoadThreshold               int    `toml:"heavy-load-threshold" json:"heavy-load-threshold"`
		HeavyLoadWaitDuration            string `toml:"heavy-load-wait-duration" json:"heavy-load-wait-duration"`
		EnableRequestBatch               bool   `toml:"enable-request-batch" json:"enable-request-batch"`
		BackgroundThreadCount            int    `toml:"background-thread-count" json:"background-thread-count"`
		EndPointSlowLogThreshold         string `toml:"end-point-slow-log-threshold" json:"end-point-slow-log-threshold"`
		ForwardMaxConnectionsPerAddress  int    `toml:"forward-max-connections-per-address" json:"forward-max-connections-per-address"`
		Labels                           struct {
			Host   string `toml:"host" json:"host"`
			Zone   string `toml:"zone" json:"zone"`
			Engine string `toml:"engine" json:"engine"`
		} `toml:"labels" json:"labels"`
	} `toml:"server" json:"server"`
	Storage struct {
		DataDir                        string  `toml:"data-dir" json:"data-dir"`
		GcRatioThreshold               float64 `toml:"gc-ratio-threshold" json:"gc-ratio-threshold"`
		MaxKeySize                     int     `toml:"max-key-size" json:"max-key-size"`
		SchedulerConcurrency           int     `toml:"scheduler-concurrency" json:"scheduler-concurrency"`
		SchedulerWorkerPoolSize        int     `toml:"scheduler-worker-pool-size" json:"scheduler-worker-pool-size"`
		SchedulerPendingWriteThreshold string  `toml:"scheduler-pending-write-threshold" json:"scheduler-pending-write-threshold"`
		ReserveSpace                   string  `toml:"reserve-space" json:"reserve-space"`
		EnableAsyncApplyPrewrite       bool    `toml:"enable-async-apply-prewrite" json:"enable-async-apply-prewrite"`
		EnableTTL                      bool    `toml:"enable-ttl" json:"enable-ttl"`
		TTLCheckPollInterval           string  `toml:"ttl-check-poll-interval" json:"ttl-check-poll-interval"`
		BlockCache                     struct {
			Shared              bool    `toml:"shared" json:"shared"`
			Capacity            string  `toml:"capacity" json:"capacity"`
			NumShardBits        int     `toml:"num-shard-bits" json:"num-shard-bits"`
			StrictCapacityLimit bool    `toml:"strict-capacity-limit" json:"strict-capacity-limit"`
			HighPriPoolRatio    float64 `toml:"high-pri-pool-ratio" json:"high-pri-pool-ratio"`
			MemoryAllocator     string  `toml:"memory-allocator" json:"memory-allocator"`
		} `toml:"block-cache" json:"block-cache"`
		IoRateLimit struct {
			MaxBytesPerSec              string `toml:"max-bytes-per-sec" json:"max-bytes-per-sec"`
			Mode                        string `toml:"mode" json:"mode"`
			Strict                      bool   `toml:"strict" json:"strict"`
			ForegroundReadPriority      string `toml:"foreground-read-priority" json:"foreground-read-priority"`
			ForegroundWritePriority     string `toml:"foreground-write-priority" json:"foreground-write-priority"`
			FlushPriority               string `toml:"flush-priority" json:"flush-priority"`
			LevelZeroCompactionPriority string `toml:"level-zero-compaction-priority" json:"level-zero-compaction-priority"`
			CompactionPriority          string `toml:"compaction-priority" json:"compaction-priority"`
			ReplicationPriority         string `toml:"replication-priority" json:"replication-priority"`
			LoadBalancePriority         string `toml:"load-balance-priority" json:"load-balance-priority"`
			GcPriority                  string `toml:"gc-priority" json:"gc-priority"`
			ImportPriority              string `toml:"import-priority" json:"import-priority"`
			ExportPriority              string `toml:"export-priority" json:"export-priority"`
			OtherPriority               string `toml:"other-priority" json:"other-priority"`
		} `toml:"io-rate-limit" json:"io-rate-limit"`
	} `toml:"storage" json:"storage"`
	Pd struct {
		Endpoints        []string `toml:"endpoints" json:"endpoints"`
		RetryInterval    string   `toml:"retry-interval" json:"retry-interval"`
		RetryMaxCount    int64    `toml:"retry-max-count" json:"retry-max-count"`
		RetryLogEvery    int      `toml:"retry-log-every" json:"retry-log-every"`
		UpdateInterval   string   `toml:"update-interval" json:"update-interval"`
		EnableForwarding bool     `toml:"enable-forwarding" json:"enable-forwarding"`
	} `toml:"pd" json:"pd"`
	Raftstore struct {
		Prevote                          bool   `toml:"prevote" json:"prevote"`
		RaftdbPath                       string `toml:"raftdb-path" json:"raftdb-path"`
		Capacity                         string `toml:"capacity" json:"capacity"`
		RaftEntryMaxSize                 string `toml:"raft-entry-max-size" json:"raft-entry-max-size"`
		RaftLogGcTickInterval            string `toml:"raft-log-gc-tick-interval" json:"raft-log-gc-tick-interval"`
		RaftLogGcThreshold               int    `toml:"raft-log-gc-threshold" json:"raft-log-gc-threshold"`
		RaftLogGcCountLimit              int    `toml:"raft-log-gc-count-limit" json:"raft-log-gc-count-limit"`
		RaftLogGcSizeLimit               string `toml:"raft-log-gc-size-limit" json:"raft-log-gc-size-limit"`
		RaftEnginePurgeInterval          string `toml:"raft-engine-purge-interval" json:"raft-engine-purge-interval"`
		RaftEntryCacheLifeTime           string `toml:"raft-entry-cache-life-time" json:"raft-entry-cache-life-time"`
		RaftRejectTransferLeaderDuration string `toml:"raft-reject-transfer-leader-duration" json:"raft-reject-transfer-leader-duration"`
		SplitRegionCheckTickInterval     string `toml:"split-region-check-tick-interval" json:"split-region-check-tick-interval"`
		RegionSplitCheckDiff             string `toml:"region-split-check-diff" json:"region-split-check-diff"`
		RegionCompactCheckInterval       string `toml:"region-compact-check-interval" json:"region-compact-check-interval"`
		RegionCompactCheckStep           int    `toml:"region-compact-check-step" json:"region-compact-check-step"`
		RegionCompactMinTombstones       int    `toml:"region-compact-min-tombstones" json:"region-compact-min-tombstones"`
		RegionCompactTombstonesPercent   int    `toml:"region-compact-tombstones-percent" json:"region-compact-tombstones-percent"`
		PdHeartbeatTickInterval          string `toml:"pd-heartbeat-tick-interval" json:"pd-heartbeat-tick-interval"`
		PdStoreHeartbeatTickInterval     string `toml:"pd-store-heartbeat-tick-interval" json:"pd-store-heartbeat-tick-interval"`
		SnapMgrGcTickInterval            string `toml:"snap-mgr-gc-tick-interval" json:"snap-mgr-gc-tick-interval"`
		SnapGcTimeout                    string `toml:"snap-gc-timeout" json:"snap-gc-timeout"`
		LockCfCompactInterval            string `toml:"lock-cf-compact-interval" json:"lock-cf-compact-interval"`
		LockCfCompactBytesThreshold      string `toml:"lock-cf-compact-bytes-threshold" json:"lock-cf-compact-bytes-threshold"`
		NotifyCapacity                   int    `toml:"notify-capacity" json:"notify-capacity"`
		MessagesPerTick                  int    `toml:"messages-per-tick" json:"messages-per-tick"`
		MaxPeerDownDuration              string `toml:"max-peer-down-duration" json:"max-peer-down-duration"`
		MaxLeaderMissingDuration         string `toml:"max-leader-missing-duration" json:"max-leader-missing-duration"`
		AbnormalLeaderMissingDuration    string `toml:"abnormal-leader-missing-duration" json:"abnormal-leader-missing-duration"`
		PeerStaleStateCheckInterval      string `toml:"peer-stale-state-check-interval" json:"peer-stale-state-check-interval"`
		SnapApplyBatchSize               string `toml:"snap-apply-batch-size" json:"snap-apply-batch-size"`
		SnapHandlePoolSize               int    `toml:"snap-handle-pool-size" json:"snap-handle-pool-size"`
		RegionWorkerTickInterval         string `toml:"region-worker-tick-interval" json:"region-worker-tick-interval"`
		ConsistencyCheckInterval         string `toml:"consistency-check-interval" json:"consistency-check-interval"`
		RaftStoreMaxLeaderLease          string `toml:"raft-store-max-leader-lease" json:"raft-store-max-leader-lease"`
		AllowRemoveLeader                bool   `toml:"allow-remove-leader" json:"allow-remove-leader"`
		MergeCheckTickInterval           string `toml:"merge-check-tick-interval" json:"merge-check-tick-interval"`
		CleanupImportSstInterval         string `toml:"cleanup-import-sst-interval" json:"cleanup-import-sst-interval"`
		LocalReadBatchSize               int    `toml:"local-read-batch-size" json:"local-read-batch-size"`
		ApplyMaxBatchSize                int    `toml:"apply-max-batch-size" json:"apply-max-batch-size"`
		ApplyPoolSize                    int    `toml:"apply-pool-size" json:"apply-pool-size"`
		ApplyRescheduleDuration          string `toml:"apply-reschedule-duration" json:"apply-reschedule-duration"`
		ApplyLowPriorityPoolSize         int    `toml:"apply-low-priority-pool-size" json:"apply-low-priority-pool-size"`
		StoreMaxBatchSize                int    `toml:"store-max-batch-size" json:"store-max-batch-size"`
		StorePoolSize                    int    `toml:"store-pool-size" json:"store-pool-size"`
		StoreRescheduleDuration          string `toml:"store-reschedule-duration" json:"store-reschedule-duration"`
		StoreLowPriorityPoolSize         int    `toml:"store-low-priority-pool-size" json:"store-low-priority-pool-size"`
		FuturePollSize                   int    `toml:"future-poll-size" json:"future-poll-size"`
		HibernateRegions                 bool   `toml:"hibernate-regions" json:"hibernate-regions"`
		PerfLevel                        int    `toml:"perf-level" json:"perf-level"`
		StoreBatchRetryRecvTimeout       string `toml:"store-batch-retry-recv-timeout" json:"store-batch-retry-recv-timeout"`
	} `toml:"raftstore" json:"raftstore"`
	Coprocessor struct {
		SplitRegionOnTable     bool   `toml:"split-region-on-table" json:"split-region-on-table"`
		BatchSplitLimit        int    `toml:"batch-split-limit" json:"batch-split-limit"`
		RegionMaxSize          string `toml:"region-max-size" json:"region-max-size"`
		RegionSplitSize        string `toml:"region-split-size" json:"region-split-size"`
		RegionMaxKeys          int    `toml:"region-max-keys" json:"region-max-keys"`
		RegionSplitKeys        int    `toml:"region-split-keys" json:"region-split-keys"`
		ConsistencyCheckMethod string `toml:"consistency-check-method" json:"consistency-check-method"`
		PerfLevel              int    `toml:"perf-level" json:"perf-level"`
	} `toml:"coprocessor" json:"coprocessor"`
	CoprocessorV2 struct {
		CoprocessorPluginDirectory interface{} `toml:"coprocessor-plugin-directory" json:"coprocessor-plugin-directory"`
	} `toml:"coprocessor-v2" json:"coprocessor-v2"`
	Rocksdb struct {
		InfoLogLevel                     string `toml:"info-log-level" json:"info-log-level"`
		WalRecoveryMode                  int    `toml:"wal-recovery-mode" json:"wal-recovery-mode"`
		WalDir                           string `toml:"wal-dir" json:"wal-dir"`
		WalTTLSeconds                    int    `toml:"wal-ttl-seconds" json:"wal-ttl-seconds"`
		WalSizeLimit                     string `toml:"wal-size-limit" json:"wal-size-limit"`
		MaxTotalWalSize                  string `toml:"max-total-wal-size" json:"max-total-wal-size"`
		MaxBackgroundJobs                int    `toml:"max-background-jobs" json:"max-background-jobs"`
		MaxBackgroundFlushes             int    `toml:"max-background-flushes" json:"max-background-flushes"`
		MaxManifestFileSize              string `toml:"max-manifest-file-size" json:"max-manifest-file-size"`
		CreateIfMissing                  bool   `toml:"create-if-missing" json:"create-if-missing"`
		MaxOpenFiles                     int    `toml:"max-open-files" json:"max-open-files"`
		EnableStatistics                 bool   `toml:"enable-statistics" json:"enable-statistics"`
		StatsDumpPeriod                  string `toml:"stats-dump-period" json:"stats-dump-period"`
		CompactionReadaheadSize          string `toml:"compaction-readahead-size" json:"compaction-readahead-size"`
		InfoLogMaxSize                   string `toml:"info-log-max-size" json:"info-log-max-size"`
		InfoLogRollTime                  string `toml:"info-log-roll-time" json:"info-log-roll-time"`
		InfoLogKeepLogFileNum            int    `toml:"info-log-keep-log-file-num" json:"info-log-keep-log-file-num"`
		InfoLogDir                       string `toml:"info-log-dir" json:"info-log-dir"`
		RateBytesPerSec                  string `toml:"rate-bytes-per-sec" json:"rate-bytes-per-sec"`
		RateLimiterRefillPeriod          string `toml:"rate-limiter-refill-period" json:"rate-limiter-refill-period"`
		RateLimiterMode                  int    `toml:"rate-limiter-mode" json:"rate-limiter-mode"`
		RateLimiterAutoTuned             bool   `toml:"rate-limiter-auto-tuned" json:"rate-limiter-auto-tuned"`
		BytesPerSync                     string `toml:"bytes-per-sync" json:"bytes-per-sync"`
		WalBytesPerSync                  string `toml:"wal-bytes-per-sync" json:"wal-bytes-per-sync"`
		MaxSubCompactions                int    `toml:"max-sub-compactions" json:"max-sub-compactions"`
		WritableFileMaxBufferSize        string `toml:"writable-file-max-buffer-size" json:"writable-file-max-buffer-size"`
		UseDirectIoForFlushAndCompaction bool   `toml:"use-direct-io-for-flush-and-compaction" json:"use-direct-io-for-flush-and-compaction"`
		EnablePipelinedWrite             bool   `toml:"enable-pipelined-write" json:"enable-pipelined-write"`
		EnableMultiBatchWrite            bool   `toml:"enable-multi-batch-write" json:"enable-multi-batch-write"`
		EnableUnorderedWrite             bool   `toml:"enable-unordered-write" json:"enable-unordered-write"`
		Defaultcf                        struct {
			BlockSize                           string   `toml:"block-size" json:"block-size"`
			BlockCacheSize                      string   `toml:"block-cache-size" json:"block-cache-size"`
			DisableBlockCache                   bool     `toml:"disable-block-cache" json:"disable-block-cache"`
			CacheIndexAndFilterBlocks           bool     `toml:"cache-index-and-filter-blocks" json:"cache-index-and-filter-blocks"`
			PinL0FilterAndIndexBlocks           bool     `toml:"pin-l0-filter-and-index-blocks" json:"pin-l0-filter-and-index-blocks"`
			UseBloomFilter                      bool     `toml:"use-bloom-filter" json:"use-bloom-filter"`
			OptimizeFiltersForHits              bool     `toml:"optimize-filters-for-hits" json:"optimize-filters-for-hits"`
			WholeKeyFiltering                   bool     `toml:"whole-key-filtering" json:"whole-key-filtering"`
			BloomFilterBitsPerKey               int      `toml:"bloom-filter-bits-per-key" json:"bloom-filter-bits-per-key"`
			BlockBasedBloomFilter               bool     `toml:"block-based-bloom-filter" json:"block-based-bloom-filter"`
			ReadAmpBytesPerBit                  int      `toml:"read-amp-bytes-per-bit" json:"read-amp-bytes-per-bit"`
			CompressionPerLevel                 []string `toml:"compression-per-level" json:"compression-per-level"`
			WriteBufferSize                     string   `toml:"write-buffer-size" json:"write-buffer-size"`
			MaxWriteBufferNumber                int      `toml:"max-write-buffer-number" json:"max-write-buffer-number"`
			MinWriteBufferNumberToMerge         int      `toml:"min-write-buffer-number-to-merge" json:"min-write-buffer-number-to-merge"`
			MaxBytesForLevelBase                string   `toml:"max-bytes-for-level-base" json:"max-bytes-for-level-base"`
			TargetFileSizeBase                  string   `toml:"target-file-size-base" json:"target-file-size-base"`
			Level0FileNumCompactionTrigger      int      `toml:"level0-file-num-compaction-trigger" json:"level0-file-num-compaction-trigger"`
			Level0SlowdownWritesTrigger         int      `toml:"level0-slowdown-writes-trigger" json:"level0-slowdown-writes-trigger"`
			Level0StopWritesTrigger             int      `toml:"level0-stop-writes-trigger" json:"level0-stop-writes-trigger"`
			MaxCompactionBytes                  string   `toml:"max-compaction-bytes" json:"max-compaction-bytes"`
			CompactionPri                       int      `toml:"compaction-pri" json:"compaction-pri"`
			DynamicLevelBytes                   bool     `toml:"dynamic-level-bytes" json:"dynamic-level-bytes"`
			NumLevels                           int      `toml:"num-levels" json:"num-levels"`
			MaxBytesForLevelMultiplier          int      `toml:"max-bytes-for-level-multiplier" json:"max-bytes-for-level-multiplier"`
			CompactionStyle                     int      `toml:"compaction-style" json:"compaction-style"`
			DisableAutoCompactions              bool     `toml:"disable-auto-compactions" json:"disable-auto-compactions"`
			SoftPendingCompactionBytesLimit     string   `toml:"soft-pending-compaction-bytes-limit" json:"soft-pending-compaction-bytes-limit"`
			HardPendingCompactionBytesLimit     string   `toml:"hard-pending-compaction-bytes-limit" json:"hard-pending-compaction-bytes-limit"`
			ForceConsistencyChecks              bool     `toml:"force-consistency-checks" json:"force-consistency-checks"`
			PropSizeIndexDistance               int      `toml:"prop-size-index-distance" json:"prop-size-index-distance"`
			PropKeysIndexDistance               int      `toml:"prop-keys-index-distance" json:"prop-keys-index-distance"`
			EnableDoublySkiplist                bool     `toml:"enable-doubly-skiplist" json:"enable-doubly-skiplist"`
			EnableCompactionGuard               bool     `toml:"enable-compaction-guard" json:"enable-compaction-guard"`
			CompactionGuardMinOutputFileSize    string   `toml:"compaction-guard-min-output-file-size" json:"compaction-guard-min-output-file-size"`
			CompactionGuardMaxOutputFileSize    string   `toml:"compaction-guard-max-output-file-size" json:"compaction-guard-max-output-file-size"`
			BottommostLevelCompression          string   `toml:"bottommost-level-compression" json:"bottommost-level-compression"`
			BottommostZstdCompressionDictSize   int      `toml:"bottommost-zstd-compression-dict-size" json:"bottommost-zstd-compression-dict-size"`
			BottommostZstdCompressionSampleSize int      `toml:"bottommost-zstd-compression-sample-size" json:"bottommost-zstd-compression-sample-size"`
			Titan                               struct {
				MinBlobSize             string  `toml:"min-blob-size" json:"min-blob-size"`
				BlobFileCompression     string  `toml:"blob-file-compression" json:"blob-file-compression"`
				BlobCacheSize           string  `toml:"blob-cache-size" json:"blob-cache-size"`
				MinGcBatchSize          string  `toml:"min-gc-batch-size" json:"min-gc-batch-size"`
				MaxGcBatchSize          string  `toml:"max-gc-batch-size" json:"max-gc-batch-size"`
				DiscardableRatio        float64 `toml:"discardable-ratio" json:"discardable-ratio"`
				SampleRatio             float64 `toml:"sample-ratio" json:"sample-ratio"`
				MergeSmallFileThreshold string  `toml:"merge-small-file-threshold" json:"merge-small-file-threshold"`
				BlobRunMode             string  `toml:"blob-run-mode" json:"blob-run-mode"`
				LevelMerge              bool    `toml:"level-merge" json:"level-merge"`
				RangeMerge              bool    `toml:"range-merge" json:"range-merge"`
				MaxSortedRuns           int     `toml:"max-sorted-runs" json:"max-sorted-runs"`
				GcMergeRewrite          bool    `toml:"gc-merge-rewrite" json:"gc-merge-rewrite"`
			} `toml:"titan" json:"titan"`
		} `toml:"defaultcf" json:"defaultcf"`
		Writecf struct {
			BlockSize                           string   `toml:"block-size" json:"block-size"`
			BlockCacheSize                      string   `toml:"block-cache-size" json:"block-cache-size"`
			DisableBlockCache                   bool     `toml:"disable-block-cache" json:"disable-block-cache"`
			CacheIndexAndFilterBlocks           bool     `toml:"cache-index-and-filter-blocks" json:"cache-index-and-filter-blocks"`
			PinL0FilterAndIndexBlocks           bool     `toml:"pin-l0-filter-and-index-blocks" json:"pin-l0-filter-and-index-blocks"`
			UseBloomFilter                      bool     `toml:"use-bloom-filter" json:"use-bloom-filter"`
			OptimizeFiltersForHits              bool     `toml:"optimize-filters-for-hits" json:"optimize-filters-for-hits"`
			WholeKeyFiltering                   bool     `toml:"whole-key-filtering" json:"whole-key-filtering"`
			BloomFilterBitsPerKey               int      `toml:"bloom-filter-bits-per-key" json:"bloom-filter-bits-per-key"`
			BlockBasedBloomFilter               bool     `toml:"block-based-bloom-filter" json:"block-based-bloom-filter"`
			ReadAmpBytesPerBit                  int      `toml:"read-amp-bytes-per-bit" json:"read-amp-bytes-per-bit"`
			CompressionPerLevel                 []string `toml:"compression-per-level" json:"compression-per-level"`
			WriteBufferSize                     string   `toml:"write-buffer-size" json:"write-buffer-size"`
			MaxWriteBufferNumber                int      `toml:"max-write-buffer-number" json:"max-write-buffer-number"`
			MinWriteBufferNumberToMerge         int      `toml:"min-write-buffer-number-to-merge" json:"min-write-buffer-number-to-merge"`
			MaxBytesForLevelBase                string   `toml:"max-bytes-for-level-base" json:"max-bytes-for-level-base"`
			TargetFileSizeBase                  string   `toml:"target-file-size-base" json:"target-file-size-base"`
			Level0FileNumCompactionTrigger      int      `toml:"level0-file-num-compaction-trigger" json:"level0-file-num-compaction-trigger"`
			Level0SlowdownWritesTrigger         int      `toml:"level0-slowdown-writes-trigger" json:"level0-slowdown-writes-trigger"`
			Level0StopWritesTrigger             int      `toml:"level0-stop-writes-trigger" json:"level0-stop-writes-trigger"`
			MaxCompactionBytes                  string   `toml:"max-compaction-bytes" json:"max-compaction-bytes"`
			CompactionPri                       int      `toml:"compaction-pri" json:"compaction-pri"`
			DynamicLevelBytes                   bool     `toml:"dynamic-level-bytes" json:"dynamic-level-bytes"`
			NumLevels                           int      `toml:"num-levels" json:"num-levels"`
			MaxBytesForLevelMultiplier          int      `toml:"max-bytes-for-level-multiplier" json:"max-bytes-for-level-multiplier"`
			CompactionStyle                     int      `toml:"compaction-style" json:"compaction-style"`
			DisableAutoCompactions              bool     `toml:"disable-auto-compactions" json:"disable-auto-compactions"`
			SoftPendingCompactionBytesLimit     string   `toml:"soft-pending-compaction-bytes-limit" json:"soft-pending-compaction-bytes-limit"`
			HardPendingCompactionBytesLimit     string   `toml:"hard-pending-compaction-bytes-limit" json:"hard-pending-compaction-bytes-limit"`
			ForceConsistencyChecks              bool     `toml:"force-consistency-checks" json:"force-consistency-checks"`
			PropSizeIndexDistance               int      `toml:"prop-size-index-distance" json:"prop-size-index-distance"`
			PropKeysIndexDistance               int      `toml:"prop-keys-index-distance" json:"prop-keys-index-distance"`
			EnableDoublySkiplist                bool     `toml:"enable-doubly-skiplist" json:"enable-doubly-skiplist"`
			EnableCompactionGuard               bool     `toml:"enable-compaction-guard" json:"enable-compaction-guard"`
			CompactionGuardMinOutputFileSize    string   `toml:"compaction-guard-min-output-file-size" json:"compaction-guard-min-output-file-size"`
			CompactionGuardMaxOutputFileSize    string   `toml:"compaction-guard-max-output-file-size" json:"compaction-guard-max-output-file-size"`
			BottommostLevelCompression          string   `toml:"bottommost-level-compression" json:"bottommost-level-compression"`
			BottommostZstdCompressionDictSize   int      `toml:"bottommost-zstd-compression-dict-size" json:"bottommost-zstd-compression-dict-size"`
			BottommostZstdCompressionSampleSize int      `toml:"bottommost-zstd-compression-sample-size" json:"bottommost-zstd-compression-sample-size"`
			Titan                               struct {
				MinBlobSize             string  `toml:"min-blob-size" json:"min-blob-size"`
				BlobFileCompression     string  `toml:"blob-file-compression" json:"blob-file-compression"`
				BlobCacheSize           string  `toml:"blob-cache-size" json:"blob-cache-size"`
				MinGcBatchSize          string  `toml:"min-gc-batch-size" json:"min-gc-batch-size"`
				MaxGcBatchSize          string  `toml:"max-gc-batch-size" json:"max-gc-batch-size"`
				DiscardableRatio        float64 `toml:"discardable-ratio" json:"discardable-ratio"`
				SampleRatio             float64 `toml:"sample-ratio" json:"sample-ratio"`
				MergeSmallFileThreshold string  `toml:"merge-small-file-threshold" json:"merge-small-file-threshold"`
				BlobRunMode             string  `toml:"blob-run-mode" json:"blob-run-mode"`
				LevelMerge              bool    `toml:"level-merge" json:"level-merge"`
				RangeMerge              bool    `toml:"range-merge" json:"range-merge"`
				MaxSortedRuns           int     `toml:"max-sorted-runs" json:"max-sorted-runs"`
				GcMergeRewrite          bool    `toml:"gc-merge-rewrite" json:"gc-merge-rewrite"`
			} `toml:"titan" json:"titan"`
		} `toml:"writecf" json:"writecf"`
		Lockcf struct {
			BlockSize                           string   `toml:"block-size" json:"block-size"`
			BlockCacheSize                      string   `toml:"block-cache-size" json:"block-cache-size"`
			DisableBlockCache                   bool     `toml:"disable-block-cache" json:"disable-block-cache"`
			CacheIndexAndFilterBlocks           bool     `toml:"cache-index-and-filter-blocks" json:"cache-index-and-filter-blocks"`
			PinL0FilterAndIndexBlocks           bool     `toml:"pin-l0-filter-and-index-blocks" json:"pin-l0-filter-and-index-blocks"`
			UseBloomFilter                      bool     `toml:"use-bloom-filter" json:"use-bloom-filter"`
			OptimizeFiltersForHits              bool     `toml:"optimize-filters-for-hits" json:"optimize-filters-for-hits"`
			WholeKeyFiltering                   bool     `toml:"whole-key-filtering" json:"whole-key-filtering"`
			BloomFilterBitsPerKey               int      `toml:"bloom-filter-bits-per-key" json:"bloom-filter-bits-per-key"`
			BlockBasedBloomFilter               bool     `toml:"block-based-bloom-filter" json:"block-based-bloom-filter"`
			ReadAmpBytesPerBit                  int      `toml:"read-amp-bytes-per-bit" json:"read-amp-bytes-per-bit"`
			CompressionPerLevel                 []string `toml:"compression-per-level" json:"compression-per-level"`
			WriteBufferSize                     string   `toml:"write-buffer-size" json:"write-buffer-size"`
			MaxWriteBufferNumber                int      `toml:"max-write-buffer-number" json:"max-write-buffer-number"`
			MinWriteBufferNumberToMerge         int      `toml:"min-write-buffer-number-to-merge" json:"min-write-buffer-number-to-merge"`
			MaxBytesForLevelBase                string   `toml:"max-bytes-for-level-base" json:"max-bytes-for-level-base"`
			TargetFileSizeBase                  string   `toml:"target-file-size-base" json:"target-file-size-base"`
			Level0FileNumCompactionTrigger      int      `toml:"level0-file-num-compaction-trigger" json:"level0-file-num-compaction-trigger"`
			Level0SlowdownWritesTrigger         int      `toml:"level0-slowdown-writes-trigger" json:"level0-slowdown-writes-trigger"`
			Level0StopWritesTrigger             int      `toml:"level0-stop-writes-trigger" json:"level0-stop-writes-trigger"`
			MaxCompactionBytes                  string   `toml:"max-compaction-bytes" json:"max-compaction-bytes"`
			CompactionPri                       int      `toml:"compaction-pri" json:"compaction-pri"`
			DynamicLevelBytes                   bool     `toml:"dynamic-level-bytes" json:"dynamic-level-bytes"`
			NumLevels                           int      `toml:"num-levels" json:"num-levels"`
			MaxBytesForLevelMultiplier          int      `toml:"max-bytes-for-level-multiplier" json:"max-bytes-for-level-multiplier"`
			CompactionStyle                     int      `toml:"compaction-style" json:"compaction-style"`
			DisableAutoCompactions              bool     `toml:"disable-auto-compactions" json:"disable-auto-compactions"`
			SoftPendingCompactionBytesLimit     string   `toml:"soft-pending-compaction-bytes-limit" json:"soft-pending-compaction-bytes-limit"`
			HardPendingCompactionBytesLimit     string   `toml:"hard-pending-compaction-bytes-limit" json:"hard-pending-compaction-bytes-limit"`
			ForceConsistencyChecks              bool     `toml:"force-consistency-checks" json:"force-consistency-checks"`
			PropSizeIndexDistance               int      `toml:"prop-size-index-distance" json:"prop-size-index-distance"`
			PropKeysIndexDistance               int      `toml:"prop-keys-index-distance" json:"prop-keys-index-distance"`
			EnableDoublySkiplist                bool     `toml:"enable-doubly-skiplist" json:"enable-doubly-skiplist"`
			EnableCompactionGuard               bool     `toml:"enable-compaction-guard" json:"enable-compaction-guard"`
			CompactionGuardMinOutputFileSize    string   `toml:"compaction-guard-min-output-file-size" json:"compaction-guard-min-output-file-size"`
			CompactionGuardMaxOutputFileSize    string   `toml:"compaction-guard-max-output-file-size" json:"compaction-guard-max-output-file-size"`
			BottommostLevelCompression          string   `toml:"bottommost-level-compression" json:"bottommost-level-compression"`
			BottommostZstdCompressionDictSize   int      `toml:"bottommost-zstd-compression-dict-size" json:"bottommost-zstd-compression-dict-size"`
			BottommostZstdCompressionSampleSize int      `toml:"bottommost-zstd-compression-sample-size" json:"bottommost-zstd-compression-sample-size"`
			Titan                               struct {
				MinBlobSize             string  `toml:"min-blob-size" json:"min-blob-size"`
				BlobFileCompression     string  `toml:"blob-file-compression" json:"blob-file-compression"`
				BlobCacheSize           string  `toml:"blob-cache-size" json:"blob-cache-size"`
				MinGcBatchSize          string  `toml:"min-gc-batch-size" json:"min-gc-batch-size"`
				MaxGcBatchSize          string  `toml:"max-gc-batch-size" json:"max-gc-batch-size"`
				DiscardableRatio        float64 `toml:"discardable-ratio" json:"discardable-ratio"`
				SampleRatio             float64 `toml:"sample-ratio" json:"sample-ratio"`
				MergeSmallFileThreshold string  `toml:"merge-small-file-threshold" json:"merge-small-file-threshold"`
				BlobRunMode             string  `toml:"blob-run-mode" json:"blob-run-mode"`
				LevelMerge              bool    `toml:"level-merge" json:"level-merge"`
				RangeMerge              bool    `toml:"range-merge" json:"range-merge"`
				MaxSortedRuns           int     `toml:"max-sorted-runs" json:"max-sorted-runs"`
				GcMergeRewrite          bool    `toml:"gc-merge-rewrite" json:"gc-merge-rewrite"`
			} `toml:"titan" json:"titan"`
		} `toml:"lockcf" json:"lockcf"`
		Raftcf struct {
			BlockSize                           string   `toml:"block-size" json:"block-size"`
			BlockCacheSize                      string   `toml:"block-cache-size" json:"block-cache-size"`
			DisableBlockCache                   bool     `toml:"disable-block-cache" json:"disable-block-cache"`
			CacheIndexAndFilterBlocks           bool     `toml:"cache-index-and-filter-blocks" json:"cache-index-and-filter-blocks"`
			PinL0FilterAndIndexBlocks           bool     `toml:"pin-l0-filter-and-index-blocks" json:"pin-l0-filter-and-index-blocks"`
			UseBloomFilter                      bool     `toml:"use-bloom-filter" json:"use-bloom-filter"`
			OptimizeFiltersForHits              bool     `toml:"optimize-filters-for-hits" json:"optimize-filters-for-hits"`
			WholeKeyFiltering                   bool     `toml:"whole-key-filtering" json:"whole-key-filtering"`
			BloomFilterBitsPerKey               int      `toml:"bloom-filter-bits-per-key" json:"bloom-filter-bits-per-key"`
			BlockBasedBloomFilter               bool     `toml:"block-based-bloom-filter" json:"block-based-bloom-filter"`
			ReadAmpBytesPerBit                  int      `toml:"read-amp-bytes-per-bit" json:"read-amp-bytes-per-bit"`
			CompressionPerLevel                 []string `toml:"compression-per-level" json:"compression-per-level"`
			WriteBufferSize                     string   `toml:"write-buffer-size" json:"write-buffer-size"`
			MaxWriteBufferNumber                int      `toml:"max-write-buffer-number" json:"max-write-buffer-number"`
			MinWriteBufferNumberToMerge         int      `toml:"min-write-buffer-number-to-merge" json:"min-write-buffer-number-to-merge"`
			MaxBytesForLevelBase                string   `toml:"max-bytes-for-level-base" json:"max-bytes-for-level-base"`
			TargetFileSizeBase                  string   `toml:"target-file-size-base" json:"target-file-size-base"`
			Level0FileNumCompactionTrigger      int      `toml:"level0-file-num-compaction-trigger" json:"level0-file-num-compaction-trigger"`
			Level0SlowdownWritesTrigger         int      `toml:"level0-slowdown-writes-trigger" json:"level0-slowdown-writes-trigger"`
			Level0StopWritesTrigger             int      `toml:"level0-stop-writes-trigger" json:"level0-stop-writes-trigger"`
			MaxCompactionBytes                  string   `toml:"max-compaction-bytes" json:"max-compaction-bytes"`
			CompactionPri                       int      `toml:"compaction-pri" json:"compaction-pri"`
			DynamicLevelBytes                   bool     `toml:"dynamic-level-bytes" json:"dynamic-level-bytes"`
			NumLevels                           int      `toml:"num-levels" json:"num-levels"`
			MaxBytesForLevelMultiplier          int      `toml:"max-bytes-for-level-multiplier" json:"max-bytes-for-level-multiplier"`
			CompactionStyle                     int      `toml:"compaction-style" json:"compaction-style"`
			DisableAutoCompactions              bool     `toml:"disable-auto-compactions" json:"disable-auto-compactions"`
			SoftPendingCompactionBytesLimit     string   `toml:"soft-pending-compaction-bytes-limit" json:"soft-pending-compaction-bytes-limit"`
			HardPendingCompactionBytesLimit     string   `toml:"hard-pending-compaction-bytes-limit" json:"hard-pending-compaction-bytes-limit"`
			ForceConsistencyChecks              bool     `toml:"force-consistency-checks" json:"force-consistency-checks"`
			PropSizeIndexDistance               int      `toml:"prop-size-index-distance" json:"prop-size-index-distance"`
			PropKeysIndexDistance               int      `toml:"prop-keys-index-distance" json:"prop-keys-index-distance"`
			EnableDoublySkiplist                bool     `toml:"enable-doubly-skiplist" json:"enable-doubly-skiplist"`
			EnableCompactionGuard               bool     `toml:"enable-compaction-guard" json:"enable-compaction-guard"`
			CompactionGuardMinOutputFileSize    string   `toml:"compaction-guard-min-output-file-size" json:"compaction-guard-min-output-file-size"`
			CompactionGuardMaxOutputFileSize    string   `toml:"compaction-guard-max-output-file-size" json:"compaction-guard-max-output-file-size"`
			BottommostLevelCompression          string   `toml:"bottommost-level-compression" json:"bottommost-level-compression"`
			BottommostZstdCompressionDictSize   int      `toml:"bottommost-zstd-compression-dict-size" json:"bottommost-zstd-compression-dict-size"`
			BottommostZstdCompressionSampleSize int      `toml:"bottommost-zstd-compression-sample-size" json:"bottommost-zstd-compression-sample-size"`
			Titan                               struct {
				MinBlobSize             string  `toml:"min-blob-size" json:"min-blob-size"`
				BlobFileCompression     string  `toml:"blob-file-compression" json:"blob-file-compression"`
				BlobCacheSize           string  `toml:"blob-cache-size" json:"blob-cache-size"`
				MinGcBatchSize          string  `toml:"min-gc-batch-size" json:"min-gc-batch-size"`
				MaxGcBatchSize          string  `toml:"max-gc-batch-size" json:"max-gc-batch-size"`
				DiscardableRatio        float64 `toml:"discardable-ratio" json:"discardable-ratio"`
				SampleRatio             float64 `toml:"sample-ratio" json:"sample-ratio"`
				MergeSmallFileThreshold string  `toml:"merge-small-file-threshold" json:"merge-small-file-threshold"`
				BlobRunMode             string  `toml:"blob-run-mode" json:"blob-run-mode"`
				LevelMerge              bool    `toml:"level-merge" json:"level-merge"`
				RangeMerge              bool    `toml:"range-merge" json:"range-merge"`
				MaxSortedRuns           int     `toml:"max-sorted-runs" json:"max-sorted-runs"`
				GcMergeRewrite          bool    `toml:"gc-merge-rewrite" json:"gc-merge-rewrite"`
			} `toml:"titan" json:"titan"`
		} `toml:"raftcf" json:"raftcf"`
		Titan struct {
			Enabled                  bool   `toml:"enabled" json:"enabled"`
			Dirname                  string `toml:"dirname" json:"dirname"`
			DisableGc                bool   `toml:"disable-gc" json:"disable-gc"`
			MaxBackgroundGc          int    `toml:"max-background-gc" json:"max-background-gc"`
			PurgeObsoleteFilesPeriod string `toml:"purge-obsolete-files-period" json:"purge-obsolete-files-period"`
		} `toml:"titan" json:"titan"`
	} `toml:"rocksdb" json:"rocksdb"`
	Raftdb struct {
		WalRecoveryMode                  int    `toml:"wal-recovery-mode" json:"wal-recovery-mode"`
		WalDir                           string `toml:"wal-dir" json:"wal-dir"`
		WalTTLSeconds                    int    `toml:"wal-ttl-seconds" json:"wal-ttl-seconds"`
		WalSizeLimit                     string `toml:"wal-size-limit" json:"wal-size-limit"`
		MaxTotalWalSize                  string `toml:"max-total-wal-size" json:"max-total-wal-size"`
		MaxBackgroundJobs                int    `toml:"max-background-jobs" json:"max-background-jobs"`
		MaxBackgroundFlushes             int    `toml:"max-background-flushes" json:"max-background-flushes"`
		MaxManifestFileSize              string `toml:"max-manifest-file-size" json:"max-manifest-file-size"`
		CreateIfMissing                  bool   `toml:"create-if-missing" json:"create-if-missing"`
		MaxOpenFiles                     int    `toml:"max-open-files" json:"max-open-files"`
		EnableStatistics                 bool   `toml:"enable-statistics" json:"enable-statistics"`
		StatsDumpPeriod                  string `toml:"stats-dump-period" json:"stats-dump-period"`
		CompactionReadaheadSize          string `toml:"compaction-readahead-size" json:"compaction-readahead-size"`
		InfoLogMaxSize                   string `toml:"info-log-max-size" json:"info-log-max-size"`
		InfoLogRollTime                  string `toml:"info-log-roll-time" json:"info-log-roll-time"`
		InfoLogKeepLogFileNum            int    `toml:"info-log-keep-log-file-num" json:"info-log-keep-log-file-num"`
		InfoLogDir                       string `toml:"info-log-dir" json:"info-log-dir"`
		InfoLogLevel                     string `toml:"info-log-level" json:"info-log-level"`
		MaxSubCompactions                int    `toml:"max-sub-compactions" json:"max-sub-compactions"`
		WritableFileMaxBufferSize        string `toml:"writable-file-max-buffer-size" json:"writable-file-max-buffer-size"`
		UseDirectIoForFlushAndCompaction bool   `toml:"use-direct-io-for-flush-and-compaction" json:"use-direct-io-for-flush-and-compaction"`
		EnablePipelinedWrite             bool   `toml:"enable-pipelined-write" json:"enable-pipelined-write"`
		EnableUnorderedWrite             bool   `toml:"enable-unordered-write" json:"enable-unordered-write"`
		AllowConcurrentMemtableWrite     bool   `toml:"allow-concurrent-memtable-write" json:"allow-concurrent-memtable-write"`
		BytesPerSync                     string `toml:"bytes-per-sync" json:"bytes-per-sync"`
		WalBytesPerSync                  string `toml:"wal-bytes-per-sync" json:"wal-bytes-per-sync"`
		Defaultcf                        struct {
			BlockSize                           string   `toml:"block-size" json:"block-size"`
			BlockCacheSize                      string   `toml:"block-cache-size" json:"block-cache-size"`
			DisableBlockCache                   bool     `toml:"disable-block-cache" json:"disable-block-cache"`
			CacheIndexAndFilterBlocks           bool     `toml:"cache-index-and-filter-blocks" json:"cache-index-and-filter-blocks"`
			PinL0FilterAndIndexBlocks           bool     `toml:"pin-l0-filter-and-index-blocks" json:"pin-l0-filter-and-index-blocks"`
			UseBloomFilter                      bool     `toml:"use-bloom-filter" json:"use-bloom-filter"`
			OptimizeFiltersForHits              bool     `toml:"optimize-filters-for-hits" json:"optimize-filters-for-hits"`
			WholeKeyFiltering                   bool     `toml:"whole-key-filtering" json:"whole-key-filtering"`
			BloomFilterBitsPerKey               int      `toml:"bloom-filter-bits-per-key" json:"bloom-filter-bits-per-key"`
			BlockBasedBloomFilter               bool     `toml:"block-based-bloom-filter" json:"block-based-bloom-filter"`
			ReadAmpBytesPerBit                  int      `toml:"read-amp-bytes-per-bit" json:"read-amp-bytes-per-bit"`
			CompressionPerLevel                 []string `toml:"compression-per-level" json:"compression-per-level"`
			WriteBufferSize                     string   `toml:"write-buffer-size" json:"write-buffer-size"`
			MaxWriteBufferNumber                int      `toml:"max-write-buffer-number" json:"max-write-buffer-number"`
			MinWriteBufferNumberToMerge         int      `toml:"min-write-buffer-number-to-merge" json:"min-write-buffer-number-to-merge"`
			MaxBytesForLevelBase                string   `toml:"max-bytes-for-level-base" json:"max-bytes-for-level-base"`
			TargetFileSizeBase                  string   `toml:"target-file-size-base" json:"target-file-size-base"`
			Level0FileNumCompactionTrigger      int      `toml:"level0-file-num-compaction-trigger" json:"level0-file-num-compaction-trigger"`
			Level0SlowdownWritesTrigger         int      `toml:"level0-slowdown-writes-trigger" json:"level0-slowdown-writes-trigger"`
			Level0StopWritesTrigger             int      `toml:"level0-stop-writes-trigger" json:"level0-stop-writes-trigger"`
			MaxCompactionBytes                  string   `toml:"max-compaction-bytes" json:"max-compaction-bytes"`
			CompactionPri                       int      `toml:"compaction-pri" json:"compaction-pri"`
			DynamicLevelBytes                   bool     `toml:"dynamic-level-bytes" json:"dynamic-level-bytes"`
			NumLevels                           int      `toml:"num-levels" json:"num-levels"`
			MaxBytesForLevelMultiplier          int      `toml:"max-bytes-for-level-multiplier" json:"max-bytes-for-level-multiplier"`
			CompactionStyle                     int      `toml:"compaction-style" json:"compaction-style"`
			DisableAutoCompactions              bool     `toml:"disable-auto-compactions" json:"disable-auto-compactions"`
			SoftPendingCompactionBytesLimit     string   `toml:"soft-pending-compaction-bytes-limit" json:"soft-pending-compaction-bytes-limit"`
			HardPendingCompactionBytesLimit     string   `toml:"hard-pending-compaction-bytes-limit" json:"hard-pending-compaction-bytes-limit"`
			ForceConsistencyChecks              bool     `toml:"force-consistency-checks" json:"force-consistency-checks"`
			PropSizeIndexDistance               int      `toml:"prop-size-index-distance" json:"prop-size-index-distance"`
			PropKeysIndexDistance               int      `toml:"prop-keys-index-distance" json:"prop-keys-index-distance"`
			EnableDoublySkiplist                bool     `toml:"enable-doubly-skiplist" json:"enable-doubly-skiplist"`
			EnableCompactionGuard               bool     `toml:"enable-compaction-guard" json:"enable-compaction-guard"`
			CompactionGuardMinOutputFileSize    string   `toml:"compaction-guard-min-output-file-size" json:"compaction-guard-min-output-file-size"`
			CompactionGuardMaxOutputFileSize    string   `toml:"compaction-guard-max-output-file-size" json:"compaction-guard-max-output-file-size"`
			BottommostLevelCompression          string   `toml:"bottommost-level-compression" json:"bottommost-level-compression"`
			BottommostZstdCompressionDictSize   int      `toml:"bottommost-zstd-compression-dict-size" json:"bottommost-zstd-compression-dict-size"`
			BottommostZstdCompressionSampleSize int      `toml:"bottommost-zstd-compression-sample-size" json:"bottommost-zstd-compression-sample-size"`
			Titan                               struct {
				MinBlobSize             string  `toml:"min-blob-size" json:"min-blob-size"`
				BlobFileCompression     string  `toml:"blob-file-compression" json:"blob-file-compression"`
				BlobCacheSize           string  `toml:"blob-cache-size" json:"blob-cache-size"`
				MinGcBatchSize          string  `toml:"min-gc-batch-size" json:"min-gc-batch-size"`
				MaxGcBatchSize          string  `toml:"max-gc-batch-size" json:"max-gc-batch-size"`
				DiscardableRatio        float64 `toml:"discardable-ratio" json:"discardable-ratio"`
				SampleRatio             float64 `toml:"sample-ratio" json:"sample-ratio"`
				MergeSmallFileThreshold string  `toml:"merge-small-file-threshold" json:"merge-small-file-threshold"`
				BlobRunMode             string  `toml:"blob-run-mode" json:"blob-run-mode"`
				LevelMerge              bool    `toml:"level-merge" json:"level-merge"`
				RangeMerge              bool    `toml:"range-merge" json:"range-merge"`
				MaxSortedRuns           int     `toml:"max-sorted-runs" json:"max-sorted-runs"`
				GcMergeRewrite          bool    `toml:"gc-merge-rewrite" json:"gc-merge-rewrite"`
			} `toml:"titan" json:"titan"`
		} `toml:"defaultcf" json:"defaultcf"`
		Titan struct {
			Enabled                  bool   `toml:"enabled" json:"enabled"`
			Dirname                  string `toml:"dirname" json:"dirname"`
			DisableGc                bool   `toml:"disable-gc" json:"disable-gc"`
			MaxBackgroundGc          int    `toml:"max-background-gc" json:"max-background-gc"`
			PurgeObsoleteFilesPeriod string `toml:"purge-obsolete-files-period" json:"purge-obsolete-files-period"`
		} `toml:"titan" json:"titan"`
	} `toml:"raftdb" json:"raftdb"`
	RaftEngine struct {
		Enable                    bool   `toml:"enable" json:"enable"`
		Dir                       string `toml:"dir" json:"dir"`
		RecoveryMode              string `toml:"recovery-mode" json:"recovery-mode"`
		BytesPerSync              string `toml:"bytes-per-sync" json:"bytes-per-sync"`
		TargetFileSize            string `toml:"target-file-size" json:"target-file-size"`
		PurgeThreshold            string `toml:"purge-threshold" json:"purge-threshold"`
		CacheLimit                string `toml:"cache-limit" json:"cache-limit"`
		BatchCompressionThreshold string `toml:"batch-compression-threshold" json:"batch-compression-threshold"`
	} `toml:"raft-engine" json:"raft-engine"`
	Security struct {
		CaPath        string        `toml:"ca-path" json:"ca-path"`
		CertPath      string        `toml:"cert-path" json:"cert-path"`
		KeyPath       string        `toml:"key-path" json:"key-path"`
		CertAllowedCn []interface{} `toml:"cert-allowed-cn" json:"cert-allowed-cn"`
		RedactInfoLog interface{}   `toml:"redact-info-log" json:"redact-info-log"`
		Encryption    struct {
			DataEncryptionMethod           string `toml:"data-encryption-method" json:"data-encryption-method"`
			DataKeyRotationPeriod          string `toml:"data-key-rotation-period" json:"data-key-rotation-period"`
			EnableFileDictionaryLog        bool   `toml:"enable-file-dictionary-log" json:"enable-file-dictionary-log"`
			FileDictionaryRewriteThreshold int    `toml:"file-dictionary-rewrite-threshold" json:"file-dictionary-rewrite-threshold"`
			MasterKey                      struct {
				Type string `toml:"type" json:"type"`
			} `toml:"master-key" json:"master-key"`
			PreviousMasterKey struct {
				Type string `toml:"type" json:"type"`
			} `toml:"previous-master-key" json:"previous-master-key"`
		} `toml:"encryption" json:"encryption"`
	} `toml:"security" json:"security"`
	Import struct {
		NumThreads          int    `toml:"num-threads" json:"num-threads"`
		StreamChannelWindow int    `toml:"stream-channel-window" json:"stream-channel-window"`
		ImportModeTimeout   string `toml:"import-mode-timeout" json:"import-mode-timeout"`
	} `toml:"import" json:"import"`
	Backup struct {
		NumThreads int    `toml:"num-threads" json:"num-threads"`
		BatchSize  int    `toml:"batch-size" json:"batch-size"`
		SstMaxSize string `toml:"sst-max-size" json:"sst-max-size"`
	} `toml:"backup" json:"backup"`
	PessimisticTxn struct {
		WaitForLockTimeout  string `toml:"wait-for-lock-timeout" json:"wait-for-lock-timeout"`
		WakeUpDelayDuration string `toml:"wake-up-delay-duration" json:"wake-up-delay-duration"`
		Pipelined           bool   `toml:"pipelined" json:"pipelined"`
	} `toml:"pessimistic-txn" json:"pessimistic-txn"`
	Gc struct {
		RatioThreshold                   float64 `toml:"ratio-threshold" json:"ratio-threshold"`
		BatchKeys                        int     `toml:"batch-keys" json:"batch-keys"`
		MaxWriteBytesPerSec              string  `toml:"max-write-bytes-per-sec" json:"max-write-bytes-per-sec"`
		EnableCompactionFilter           bool    `toml:"enable-compaction-filter" json:"enable-compaction-filter"`
		CompactionFilterSkipVersionCheck bool    `toml:"compaction-filter-skip-version-check" json:"compaction-filter-skip-version-check"`
	} `toml:"gc" json:"gc"`
	Split struct {
		QPSThreshold        int     `toml:"qps-threshold" json:"qps-threshold"`
		SplitBalanceScore   float64 `toml:"split-balance-score" json:"split-balance-score"`
		SplitContainedScore float64 `toml:"split-contained-score" json:"split-contained-score"`
		DetectTimes         int     `toml:"detect-times" json:"detect-times"`
		SampleNum           int     `toml:"sample-num" json:"sample-num"`
		SampleThreshold     int     `toml:"sample-threshold" json:"sample-threshold"`
		ByteThreshold       int     `toml:"byte-threshold" json:"byte-threshold"`
	} `toml:"split" json:"split"`
	Cdc struct {
		MinTsInterval              string `toml:"min-ts-interval" json:"min-ts-interval"`
		HibernateRegionsCompatible bool   `toml:"hibernate-regions-compatible" json:"hibernate-regions-compatible"`
		IncrementalScanThreads     int    `toml:"incremental-scan-threads" json:"incremental-scan-threads"`
		IncrementalScanConcurrency int    `toml:"incremental-scan-concurrency" json:"incremental-scan-concurrency"`
		IncrementalScanSpeedLimit  string `toml:"incremental-scan-speed-limit" json:"incremental-scan-speed-limit"`
		SinkMemoryQuota            string `toml:"sink-memory-quota" json:"sink-memory-quota"`
		OldValueCacheMemoryQuota   string `toml:"old-value-cache-memory-quota" json:"old-value-cache-memory-quota"`
		OldValueCacheSize          int    `toml:"old-value-cache-size" json:"old-value-cache-size"`
	} `toml:"cdc" json:"cdc"`
	ResolvedTs struct {
		Enable            bool   `toml:"enable" json:"enable"`
		AdvanceTsInterval string `toml:"advance-ts-interval" json:"advance-ts-interval"`
		ScanLockPoolSize  int    `toml:"scan-lock-pool-size" json:"scan-lock-pool-size"`
	} `toml:"resolved-ts" json:"resolved-ts"`
	ResourceMetering struct {
		Enabled             bool   `toml:"enabled" json:"enabled"`
		AgentAddress        string `toml:"agent-address" json:"agent-address"`
		ReportAgentInterval string `toml:"report-agent-interval" json:"report-agent-interval"`
		MaxResourceGroups   int    `toml:"max-resource-groups" json:"max-resource-groups"`
		Precision           string `toml:"precision" json:"precision"`
	} `toml:"resource-metering" json:"resource-metering"`
}

type TiflashConfigData struct {
	*TiflashConfig
	Port int // TODO move to meta
}

func (cfg *TiflashConfigData) GetValueByTagPath(tagPath string) reflect.Value {
	tags := strings.Split(tagPath, ".")
	if len(tags) == 0 {
		return reflect.ValueOf(cfg.TiflashConfig)
	}
	value := utils.VisitByTagPath(reflect.ValueOf(cfg.TiflashConfig), tags, 0)
	return value
}

func (cfg *TiflashConfigData) GetComponent() string {
	return TiflashComponentName
}

func (cfg *TiflashConfigData) GetPort() int {
	return cfg.Port
}

func NewTiflashConfigData() *TiflashConfigData {
	return &TiflashConfigData{
		TiflashConfig: &TiflashConfig{},
	}
}
