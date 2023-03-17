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
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/joomcode/errorx"
	json "github.com/json-iterator/go"
	"github.com/klauspost/compress/zstd"
	"github.com/pingcap/diag/pkg/models"
	"github.com/pingcap/diag/pkg/utils"
	perrs "github.com/pingcap/errors"
	"github.com/pingcap/tiup/pkg/cluster/ctxt"
	operator "github.com/pingcap/tiup/pkg/cluster/operation"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/cluster/task"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
	"github.com/pingcap/tiup/pkg/set"
	"github.com/pingcap/tiup/pkg/tui/progress"
	tiuputils "github.com/pingcap/tiup/pkg/utils"
)

const (
	subdirMonitor = "monitor"
	subdirAlerts  = "alerts"
	subdirMetrics = "metrics"
	subdirRaw     = "raw"
	maxQueryRange = 120 * 60 // 120min
	minQueryRange = 5 * 60   // 5min
)

type collectMonitor struct {
	Metric bool
	Alert  bool
}

// AlertCollectOptions is the options collecting alerts
type AlertCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	resultDir string
	compress  bool
}

// Desc implements the Collector interface
func (c *AlertCollectOptions) Desc() string {
	return "alert lists from Prometheus node"
}

// GetBaseOptions implements the Collector interface
func (c *AlertCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *AlertCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *AlertCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *AlertCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *AlertCollectOptions) Prepare(_ *Manager, _ *models.TiDBCluster) (map[string][]CollectStat, error) {
	return nil, nil
}

// Collect implements the Collector interface
func (c *AlertCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	if m.mode != CollectModeManual && len(topo.Monitors) < 1 {
		fmt.Println("No monitoring node (prometheus) found in topology, skip.")
		return nil
	}

	monitors := make([]string, 0)
	if eps, found := topo.Attributes[AttrKeyPromEndpoint]; found {
		monitors = append(monitors, eps.([]string)...)
	} else {
		for _, prom := range topo.Monitors {
			monitors = append(monitors, fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort()))
		}
	}

	var queryOK bool
	var queryErr error
	for _, promAddr := range monitors {
		if err := ensureMonitorDir(c.resultDir, subdirAlerts, strings.ReplaceAll(promAddr, ":", "-")); err != nil {
			return err
		}

		client := &http.Client{Timeout: time.Second * 10}
		resp, err := client.PostForm(fmt.Sprintf("http://%s/api/v1/query", promAddr), url.Values{"query": {"ALERTS"}})
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		f, err := os.Create(filepath.Join(c.resultDir, subdirMonitor, subdirAlerts, strings.ReplaceAll(promAddr, ":", "-"), "alerts.json"))
		if err == nil {
			queryOK = true
		} else {
			queryErr = err
		}
		defer f.Close()

		var enc io.WriteCloser
		if c.compress {
			enc, err = zstd.NewWriter(f)
			if err != nil {
				m.logger.Errorf("failed compressing alert list: %s, retry...\n", err)
				return err
			}
			defer enc.Close()
		} else {
			enc = f
		}
		_, err = io.Copy(enc, resp.Body)
		if err != nil {
			m.logger.Errorf("failed writing alert list to file: %s, retry...\n", err)
			return err
		}
	}

	// if query successed for any one of prometheus, ignore errors for other instances
	if !queryOK {
		return queryErr
	}
	return nil
}

// MetricCollectOptions is the options collecting metrics
type MetricCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	resultDir string
	label     map[string]string
	metrics   []string // metric list
	filter    []string
	limit     int // series*min per query
	compress  bool
}

// Desc implements the Collector interface
func (c *MetricCollectOptions) Desc() string {
	return "metrics from Prometheus node"
}

// GetBaseOptions implements the Collector interface
func (c *MetricCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *MetricCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *MetricCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *MetricCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *MetricCollectOptions) Prepare(m *Manager, topo *models.TiDBCluster) (map[string][]CollectStat, error) {
	if m.mode != CollectModeManual && len(topo.Monitors) < 1 {
		if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
			fmt.Println("No Prometheus node found in topology, skip.")
		} else {
			m.logger.Warnf("No Prometheus node found in topology, skip.")
		}
		return nil, nil
	}

	tsEnd, _ := utils.ParseTime(c.GetBaseOptions().ScrapeEnd)
	tsStart, _ := utils.ParseTime(c.GetBaseOptions().ScrapeBegin)
	nsec := tsEnd.Unix() - tsStart.Unix()

	monitors := make([]string, 0)
	if eps, found := topo.Attributes[AttrKeyPromEndpoint]; found {
		monitors = append(monitors, eps.([]string)...)
	} else {
		for _, prom := range topo.Monitors {
			monitors = append(monitors, fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort()))
		}
	}

	var queryErr error
	var promAddr string
	for _, prom := range monitors {
		promAddr = prom
		client := &http.Client{Timeout: time.Second * 10}
		c.metrics, queryErr = getMetricList(client, promAddr)
		if queryErr == nil {
			break
		}
	}
	// if query successed for any one of prometheus, ignore errors for other instances
	if queryErr != nil {
		return nil, queryErr
	}

	c.metrics = filterMetrics(c.metrics, c.filter)

	result := make(map[string][]CollectStat)
	insCnt := len(topo.Components())
	cStat := CollectStat{
		Target: fmt.Sprintf("%d metrics, compressed", len(c.metrics)),
		Size:   int64(11*len(c.metrics)*insCnt) * nsec, // empirical formula, inaccurate
	}
	// compression rate is approximately 2.5%
	cStat.Size = int64(float64(cStat.Size) * 0.025)

	result[promAddr] = append(result[promAddr], cStat)

	return result, nil
}

// Collect implements the Collector interface
func (c *MetricCollectOptions) Collect(m *Manager, topo *models.TiDBCluster) error {
	if m.mode != CollectModeManual && len(topo.Monitors) < 1 {
		if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
			fmt.Println("No Prometheus node found in topology, skip.")
		} else {
			m.logger.Warnf("No Prometheus node found in topology, skip.")
		}
		return nil
	}

	monitors := make([]string, 0)
	if eps, found := topo.Attributes[AttrKeyPromEndpoint]; found {
		monitors = append(monitors, eps.([]string)...)
	} else {
		for _, prom := range topo.Monitors {
			monitors = append(monitors, fmt.Sprintf("%s:%d", prom.Host(), prom.MainPort()))
		}
	}

	mb := progress.NewMultiBar("+ Dumping metrics")
	bars := make(map[string]*progress.MultiBarItem)
	total := len(c.metrics)
	mu := sync.Mutex{}
	for _, prom := range monitors {
		key := prom
		if _, ok := bars[key]; !ok {
			bars[key] = mb.AddBar(fmt.Sprintf("  - Querying server %s", key))
		}
	}
	switch m.mode {
	case CollectModeTiUP,
		CollectModeManual:
		mb.StartRenderLoop()
		defer mb.StopRenderLoop()
	}

	qLimit := c.opt.Concurrency
	cpuCnt := runtime.NumCPU()
	if cpuCnt < qLimit {
		qLimit = cpuCnt
	}
	tl := utils.NewTokenLimiter(uint(qLimit))

	for _, prom := range monitors {
		key := prom
		done := 1

		if err := ensureMonitorDir(c.resultDir, subdirMetrics, strings.ReplaceAll(prom, ":", "-")); err != nil {
			bars[key].UpdateDisplay(&progress.DisplayProps{
				Prefix: fmt.Sprintf("  - Query server %s: %s", key, err),
				Mode:   progress.ModeError,
			})
			return err
		}

		client := &http.Client{Timeout: time.Second * 60}

		for _, mtc := range c.metrics {
			go func(tok *utils.Token, mtc string) {
				bars[key].UpdateDisplay(&progress.DisplayProps{
					Prefix: fmt.Sprintf("  - Querying server %s", key),
					Suffix: fmt.Sprintf("%d/%d querying %s ...", done, total, mtc),
				})

				tsEnd, _ := utils.ParseTime(c.GetBaseOptions().ScrapeEnd)
				tsStart, _ := utils.ParseTime(c.GetBaseOptions().ScrapeBegin)
				collectMetric(m.logger, client, prom, tsStart, tsEnd, mtc, c.label, c.resultDir, c.limit, c.compress)

				mu.Lock()
				done++
				if done >= total {
					bars[key].UpdateDisplay(&progress.DisplayProps{
						Prefix: fmt.Sprintf("  - Query server %s", key),
						Mode:   progress.ModeDone,
					})
				}
				mu.Unlock()

				tl.Put(tok)
			}(tl.Get(), mtc)
		}
	}

	tl.Wait()

	return nil
}

func getMetricList(c *http.Client, prom string) ([]string, error) {
	/*
		resp, err := c.Get(fmt.Sprintf("http://%s/api/v1/label/__name__/values", prom))
		if err != nil {
			return []string{}, err
		}
		defer resp.Body.Close()

		r := struct {
			Metrics []string `json:"data"`
		}{}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return []string{}, err
		}
		return r.Metrics, nil
		fmt.Println(strings.Join(r.Metrics, "\",\n\""))
	*/
	return []string{
		"br_raw_backup_region_seconds_bucket",
		"br_raw_backup_region_seconds_count",
		"br_raw_backup_region_seconds_sum",
		"etcd_cluster_version",
		"etcd_debugging_auth_revision",
		"etcd_debugging_disk_backend_commit_rebalance_duration_seconds_bucket",
		"etcd_debugging_disk_backend_commit_rebalance_duration_seconds_count",
		"etcd_debugging_disk_backend_commit_rebalance_duration_seconds_sum",
		"etcd_debugging_disk_backend_commit_spill_duration_seconds_bucket",
		"etcd_debugging_disk_backend_commit_spill_duration_seconds_count",
		"etcd_debugging_disk_backend_commit_spill_duration_seconds_sum",
		"etcd_debugging_disk_backend_commit_write_duration_seconds_bucket",
		"etcd_debugging_disk_backend_commit_write_duration_seconds_count",
		"etcd_debugging_disk_backend_commit_write_duration_seconds_sum",
		"etcd_debugging_lease_granted_total",
		"etcd_debugging_lease_renewed_total",
		"etcd_debugging_lease_revoked_total",
		"etcd_debugging_lease_ttl_total_bucket",
		"etcd_debugging_lease_ttl_total_count",
		"etcd_debugging_lease_ttl_total_sum",
		"etcd_debugging_mvcc_compact_revision",
		"etcd_debugging_mvcc_current_revision",
		"etcd_debugging_mvcc_db_compaction_keys_total",
		"etcd_debugging_mvcc_db_compaction_pause_duration_milliseconds_bucket",
		"etcd_debugging_mvcc_db_compaction_pause_duration_milliseconds_count",
		"etcd_debugging_mvcc_db_compaction_pause_duration_milliseconds_sum",
		"etcd_debugging_mvcc_db_compaction_total_duration_milliseconds_bucket",
		"etcd_debugging_mvcc_db_compaction_total_duration_milliseconds_count",
		"etcd_debugging_mvcc_db_compaction_total_duration_milliseconds_sum",
		"etcd_debugging_mvcc_db_total_size_in_bytes",
		"etcd_debugging_mvcc_delete_total",
		"etcd_debugging_mvcc_events_total",
		"etcd_debugging_mvcc_index_compaction_pause_duration_milliseconds_bucket",
		"etcd_debugging_mvcc_index_compaction_pause_duration_milliseconds_count",
		"etcd_debugging_mvcc_index_compaction_pause_duration_milliseconds_sum",
		"etcd_debugging_mvcc_keys_total",
		"etcd_debugging_mvcc_pending_events_total",
		"etcd_debugging_mvcc_put_total",
		"etcd_debugging_mvcc_range_total",
		"etcd_debugging_mvcc_slow_watcher_total",
		"etcd_debugging_mvcc_total_put_size_in_bytes",
		"etcd_debugging_mvcc_txn_total",
		"etcd_debugging_mvcc_watch_stream_total",
		"etcd_debugging_mvcc_watcher_total",
		"etcd_debugging_server_lease_expired_total",
		"etcd_debugging_snap_save_marshalling_duration_seconds_bucket",
		"etcd_debugging_snap_save_marshalling_duration_seconds_count",
		"etcd_debugging_snap_save_marshalling_duration_seconds_sum",
		"etcd_debugging_snap_save_total_duration_seconds_bucket",
		"etcd_debugging_snap_save_total_duration_seconds_count",
		"etcd_debugging_snap_save_total_duration_seconds_sum",
		"etcd_debugging_store_expires_total",
		"etcd_debugging_store_reads_total",
		"etcd_debugging_store_watch_requests_total",
		"etcd_debugging_store_watchers",
		"etcd_debugging_store_writes_total",
		"etcd_disk_backend_commit_duration_seconds_bucket",
		"etcd_disk_backend_commit_duration_seconds_count",
		"etcd_disk_backend_commit_duration_seconds_sum",
		"etcd_disk_backend_defrag_duration_seconds_bucket",
		"etcd_disk_backend_defrag_duration_seconds_count",
		"etcd_disk_backend_defrag_duration_seconds_sum",
		"etcd_disk_backend_snapshot_duration_seconds_bucket",
		"etcd_disk_backend_snapshot_duration_seconds_count",
		"etcd_disk_backend_snapshot_duration_seconds_sum",
		"etcd_disk_defrag_inflight",
		"etcd_disk_wal_fsync_duration_seconds_bucket",
		"etcd_disk_wal_fsync_duration_seconds_count",
		"etcd_disk_wal_fsync_duration_seconds_sum",
		"etcd_disk_wal_write_bytes_total",
		"etcd_mvcc_db_open_read_transactions",
		"etcd_mvcc_db_total_size_in_bytes",
		"etcd_mvcc_db_total_size_in_use_in_bytes",
		"etcd_mvcc_delete_total",
		"etcd_mvcc_hash_duration_seconds_bucket",
		"etcd_mvcc_hash_duration_seconds_count",
		"etcd_mvcc_hash_duration_seconds_sum",
		"etcd_mvcc_hash_rev_duration_seconds_bucket",
		"etcd_mvcc_hash_rev_duration_seconds_count",
		"etcd_mvcc_hash_rev_duration_seconds_sum",
		"etcd_mvcc_put_total",
		"etcd_mvcc_range_total",
		"etcd_mvcc_txn_total",
		"etcd_network_client_grpc_received_bytes_total",
		"etcd_network_client_grpc_sent_bytes_total",
		"etcd_server_client_requests_total",
		"etcd_server_go_version",
		"etcd_server_has_leader",
		"etcd_server_health_failures",
		"etcd_server_health_success",
		"etcd_server_heartbeat_send_failures_total",
		"etcd_server_id",
		"etcd_server_is_leader",
		"etcd_server_is_learner",
		"etcd_server_leader_changes_seen_total",
		"etcd_server_learner_promote_successes",
		"etcd_server_proposals_applied_total",
		"etcd_server_proposals_committed_total",
		"etcd_server_proposals_failed_total",
		"etcd_server_proposals_pending",
		"etcd_server_quota_backend_bytes",
		"etcd_server_read_indexes_failed_total",
		"etcd_server_slow_apply_total",
		"etcd_server_slow_read_indexes_total",
		"etcd_server_snapshot_apply_in_progress_total",
		"etcd_server_version",
		"etcd_snap_db_fsync_duration_seconds_bucket",
		"etcd_snap_db_fsync_duration_seconds_count",
		"etcd_snap_db_fsync_duration_seconds_sum",
		"etcd_snap_db_save_total_duration_seconds_bucket",
		"etcd_snap_db_save_total_duration_seconds_count",
		"etcd_snap_db_save_total_duration_seconds_sum",
		"etcd_snap_fsync_duration_seconds_bucket",
		"etcd_snap_fsync_duration_seconds_count",
		"etcd_snap_fsync_duration_seconds_sum",
		"exposer_request_latencies",
		"exposer_request_latencies_count",
		"exposer_request_latencies_sum",
		"exposer_scrapes_total",
		"exposer_transferred_bytes_total",
		"go_cgo_go_to_c_calls_calls_total",
		"go_gc_cycles_automatic_gc_cycles_total",
		"go_gc_cycles_forced_gc_cycles_total",
		"go_gc_cycles_total_gc_cycles_total",
		"go_gc_duration_seconds",
		"go_gc_duration_seconds_count",
		"go_gc_duration_seconds_sum",
		"go_gc_heap_allocs_by_size_bytes_bucket",
		"go_gc_heap_allocs_by_size_bytes_count",
		"go_gc_heap_allocs_by_size_bytes_sum",
		"go_gc_heap_allocs_bytes_total",
		"go_gc_heap_allocs_objects_total",
		"go_gc_heap_frees_by_size_bytes_bucket",
		"go_gc_heap_frees_by_size_bytes_count",
		"go_gc_heap_frees_by_size_bytes_sum",
		"go_gc_heap_frees_bytes_total",
		"go_gc_heap_frees_objects_total",
		"go_gc_heap_goal_bytes",
		"go_gc_heap_objects_objects",
		"go_gc_heap_tiny_allocs_objects_total",
		"go_gc_limiter_last_enabled_gc_cycle",
		"go_gc_pauses_seconds_bucket",
		"go_gc_pauses_seconds_count",
		"go_gc_pauses_seconds_sum",
		"go_gc_stack_starting_size_bytes",
		"go_goroutines",
		"go_info",
		"go_memory_classes_heap_free_bytes",
		"go_memory_classes_heap_objects_bytes",
		"go_memory_classes_heap_released_bytes",
		"go_memory_classes_heap_stacks_bytes",
		"go_memory_classes_heap_unused_bytes",
		"go_memory_classes_metadata_mcache_free_bytes",
		"go_memory_classes_metadata_mcache_inuse_bytes",
		"go_memory_classes_metadata_mspan_free_bytes",
		"go_memory_classes_metadata_mspan_inuse_bytes",
		"go_memory_classes_metadata_other_bytes",
		"go_memory_classes_os_stacks_bytes",
		"go_memory_classes_other_bytes",
		"go_memory_classes_profiling_buckets_bytes",
		"go_memory_classes_total_bytes",
		"go_memstats_alloc_bytes",
		"go_memstats_alloc_bytes_total",
		"go_memstats_buck_hash_sys_bytes",
		"go_memstats_frees_total",
		"go_memstats_gc_cpu_fraction",
		"go_memstats_gc_sys_bytes",
		"go_memstats_heap_alloc_bytes",
		"go_memstats_heap_idle_bytes",
		"go_memstats_heap_inuse_bytes",
		"go_memstats_heap_objects",
		"go_memstats_heap_released_bytes",
		"go_memstats_heap_sys_bytes",
		"go_memstats_last_gc_time_seconds",
		"go_memstats_lookups_total",
		"go_memstats_mallocs_total",
		"go_memstats_mcache_inuse_bytes",
		"go_memstats_mcache_sys_bytes",
		"go_memstats_mspan_inuse_bytes",
		"go_memstats_mspan_sys_bytes",
		"go_memstats_next_gc_bytes",
		"go_memstats_other_sys_bytes",
		"go_memstats_stack_inuse_bytes",
		"go_memstats_stack_sys_bytes",
		"go_memstats_sys_bytes",
		"go_sched_gomaxprocs_threads",
		"go_sched_goroutines_goroutines",
		"go_sched_latencies_seconds_bucket",
		"go_sched_latencies_seconds_count",
		"go_sched_latencies_seconds_sum",
		"go_threads",
		"grpc_server_handled_total",
		"grpc_server_handling_seconds_bucket",
		"grpc_server_handling_seconds_count",
		"grpc_server_handling_seconds_sum",
		"grpc_server_msg_received_total",
		"grpc_server_msg_sent_total",
		"grpc_server_started_total",
		"net_conntrack_dialer_conn_attempted_total",
		"net_conntrack_dialer_conn_closed_total",
		"net_conntrack_dialer_conn_established_total",
		"net_conntrack_dialer_conn_failed_total",
		"net_conntrack_listener_conn_accepted_total",
		"net_conntrack_listener_conn_closed_total",
		"os_fd_limit",
		"os_fd_used",
		"pd_checker_event_count",
		"pd_checker_patrol_regions_time",
		"pd_checker_region_list",
		"pd_client_cmd_handle_cmds_duration_seconds_bucket",
		"pd_client_cmd_handle_cmds_duration_seconds_count",
		"pd_client_cmd_handle_cmds_duration_seconds_sum",
		"pd_client_cmd_handle_failed_cmds_duration_seconds_bucket",
		"pd_client_cmd_handle_failed_cmds_duration_seconds_count",
		"pd_client_cmd_handle_failed_cmds_duration_seconds_sum",
		"pd_client_request_handle_requests_duration_seconds_bucket",
		"pd_client_request_handle_requests_duration_seconds_count",
		"pd_client_request_handle_requests_duration_seconds_sum",
		"pd_client_request_handle_tso_batch_size_bucket",
		"pd_client_request_handle_tso_batch_size_count",
		"pd_client_request_handle_tso_batch_size_sum",
		"pd_client_request_handle_tso_best_batch_size_bucket",
		"pd_client_request_handle_tso_best_batch_size_count",
		"pd_client_request_handle_tso_best_batch_size_sum",
		"pd_client_request_tso_batch_send_latency_bucket",
		"pd_client_request_tso_batch_send_latency_count",
		"pd_client_request_tso_batch_send_latency_sum",
		"pd_cluster_health_status",
		"pd_cluster_id",
		"pd_cluster_metadata",
		"pd_cluster_region_event",
		"pd_cluster_status",
		"pd_cluster_store_limit",
		"pd_cluster_store_sync",
		"pd_cluster_tso",
		"pd_cluster_tso_gap_millionseconds",
		"pd_cluster_update_stores_stats_time",
		"pd_config_status",
		"pd_hbstream_region_message",
		"pd_hotcache_flow_queue_status",
		"pd_hotcache_status",
		"pd_hotspot_status",
		"pd_monitor_time_jump_back_total",
		"pd_region_syncer_status",
		"pd_regions_abnormal_peer_duration_seconds_bucket",
		"pd_regions_abnormal_peer_duration_seconds_count",
		"pd_regions_abnormal_peer_duration_seconds_sum",
		"pd_regions_label_level",
		"pd_regions_offline_status",
		"pd_regions_status",
		"pd_schedule_filter",
		"pd_schedule_finish_operator_steps_duration_seconds_bucket",
		"pd_schedule_finish_operator_steps_duration_seconds_count",
		"pd_schedule_finish_operator_steps_duration_seconds_sum",
		"pd_schedule_finish_operators_duration_seconds_bucket",
		"pd_schedule_finish_operators_duration_seconds_count",
		"pd_schedule_finish_operators_duration_seconds_sum",
		"pd_schedule_operator_region_size_bucket",
		"pd_schedule_operator_region_size_count",
		"pd_schedule_operator_region_size_sum",
		"pd_schedule_operators_count",
		"pd_schedule_operators_waiting_count",
		"pd_schedule_waiting_operators_duration_seconds_bucket",
		"pd_schedule_waiting_operators_duration_seconds_count",
		"pd_schedule_waiting_operators_duration_seconds_sum",
		"pd_scheduler_buckets_hot_degree_hist_bucket",
		"pd_scheduler_buckets_hot_degree_hist_count",
		"pd_scheduler_buckets_hot_degree_hist_sum",
		"pd_scheduler_event_count",
		"pd_scheduler_handle_region_heartbeat_duration_seconds_bucket",
		"pd_scheduler_handle_region_heartbeat_duration_seconds_count",
		"pd_scheduler_handle_region_heartbeat_duration_seconds_sum",
		"pd_scheduler_handle_store_heartbeat_duration_seconds_bucket",
		"pd_scheduler_handle_store_heartbeat_duration_seconds_count",
		"pd_scheduler_handle_store_heartbeat_duration_seconds_sum",
		"pd_scheduler_hot_peers_summary",
		"pd_scheduler_hot_region",
		"pd_scheduler_read_byte_hist_bucket",
		"pd_scheduler_read_byte_hist_count",
		"pd_scheduler_read_byte_hist_sum",
		"pd_scheduler_read_key_hist_bucket",
		"pd_scheduler_read_key_hist_count",
		"pd_scheduler_read_key_hist_sum",
		"pd_scheduler_region_heartbeat",
		"pd_scheduler_region_heartbeat_interval_hist_bucket",
		"pd_scheduler_region_heartbeat_interval_hist_count",
		"pd_scheduler_region_heartbeat_interval_hist_sum",
		"pd_scheduler_region_heartbeat_latency_seconds_bucket",
		"pd_scheduler_region_heartbeat_latency_seconds_count",
		"pd_scheduler_region_heartbeat_latency_seconds_sum",
		"pd_scheduler_status",
		"pd_scheduler_store_heartbeat_interval_hist_bucket",
		"pd_scheduler_store_heartbeat_interval_hist_count",
		"pd_scheduler_store_heartbeat_interval_hist_sum",
		"pd_scheduler_store_status",
		"pd_scheduler_write_byte_hist_bucket",
		"pd_scheduler_write_byte_hist_count",
		"pd_scheduler_write_byte_hist_sum",
		"pd_scheduler_write_key_hist_bucket",
		"pd_scheduler_write_key_hist_count",
		"pd_scheduler_write_key_hist_sum",
		"pd_server_cluster_state_cpu_usage",
		"pd_server_etcd_state",
		"pd_server_handle_tso_duration_seconds_bucket",
		"pd_server_handle_tso_duration_seconds_count",
		"pd_server_handle_tso_duration_seconds_sum",
		"pd_server_handle_tso_proxy_batch_size_bucket",
		"pd_server_handle_tso_proxy_batch_size_count",
		"pd_server_handle_tso_proxy_batch_size_sum",
		"pd_server_handle_tso_proxy_duration_seconds_bucket",
		"pd_server_handle_tso_proxy_duration_seconds_count",
		"pd_server_handle_tso_proxy_duration_seconds_sum",
		"pd_server_info",
		"pd_service_audit_handling_seconds_bucket",
		"pd_service_audit_handling_seconds_count",
		"pd_service_audit_handling_seconds_sum",
		"pd_tso_events",
		"pd_tso_role",
		"pd_txn_handle_txns_duration_seconds_bucket",
		"pd_txn_handle_txns_duration_seconds_count",
		"pd_txn_handle_txns_duration_seconds_sum",
		"pd_txn_txns_count",
		"process_cpu_seconds_total",
		"process_max_fds",
		"process_open_fds",
		"process_resident_memory_bytes",
		"process_start_time_seconds",
		"process_virtual_memory_bytes",
		"process_virtual_memory_max_bytes",
		"prometheus_api_remote_read_queries",
		"prometheus_build_info",
		"prometheus_config_last_reload_success_timestamp_seconds",
		"prometheus_config_last_reload_successful",
		"prometheus_engine_queries",
		"prometheus_engine_queries_concurrent_max",
		"prometheus_engine_query_duration_seconds",
		"prometheus_engine_query_duration_seconds_count",
		"prometheus_engine_query_duration_seconds_sum",
		"prometheus_engine_query_log_enabled",
		"prometheus_engine_query_log_failures_total",
		"prometheus_http_request_duration_seconds_bucket",
		"prometheus_http_request_duration_seconds_count",
		"prometheus_http_request_duration_seconds_sum",
		"prometheus_http_requests_total",
		"prometheus_http_response_size_bytes_bucket",
		"prometheus_http_response_size_bytes_count",
		"prometheus_http_response_size_bytes_sum",
		"prometheus_notifications_alertmanagers_discovered",
		"prometheus_notifications_dropped_total",
		"prometheus_notifications_queue_capacity",
		"prometheus_notifications_queue_length",
		"prometheus_remote_storage_exemplars_in_total",
		"prometheus_remote_storage_highest_timestamp_in_seconds",
		"prometheus_remote_storage_samples_in_total",
		"prometheus_remote_storage_string_interner_zero_reference_releases_total",
		"prometheus_rule_evaluation_duration_seconds",
		"prometheus_rule_evaluation_duration_seconds_count",
		"prometheus_rule_evaluation_duration_seconds_sum",
		"prometheus_rule_group_duration_seconds",
		"prometheus_rule_group_duration_seconds_count",
		"prometheus_rule_group_duration_seconds_sum",
		"prometheus_sd_consul_rpc_duration_seconds",
		"prometheus_sd_consul_rpc_duration_seconds_count",
		"prometheus_sd_consul_rpc_duration_seconds_sum",
		"prometheus_sd_consul_rpc_failures_total",
		"prometheus_sd_discovered_targets",
		"prometheus_sd_dns_lookup_failures_total",
		"prometheus_sd_dns_lookups_total",
		"prometheus_sd_failed_configs",
		"prometheus_sd_file_mtime_seconds",
		"prometheus_sd_file_read_errors_total",
		"prometheus_sd_file_scan_duration_seconds",
		"prometheus_sd_file_scan_duration_seconds_count",
		"prometheus_sd_file_scan_duration_seconds_sum",
		"prometheus_sd_kubernetes_events_total",
		"prometheus_sd_received_updates_total",
		"prometheus_sd_updates_total",
		"prometheus_target_interval_length_seconds",
		"prometheus_target_interval_length_seconds_count",
		"prometheus_target_interval_length_seconds_sum",
		"prometheus_target_metadata_cache_bytes",
		"prometheus_target_metadata_cache_entries",
		"prometheus_target_scrape_pool_exceeded_label_limits_total",
		"prometheus_target_scrape_pool_exceeded_target_limit_total",
		"prometheus_target_scrape_pool_reloads_failed_total",
		"prometheus_target_scrape_pool_reloads_total",
		"prometheus_target_scrape_pool_sync_total",
		"prometheus_target_scrape_pool_targets",
		"prometheus_target_scrape_pools_failed_total",
		"prometheus_target_scrape_pools_total",
		"prometheus_target_scrapes_cache_flush_forced_total",
		"prometheus_target_scrapes_exceeded_sample_limit_total",
		"prometheus_target_scrapes_exemplar_out_of_order_total",
		"prometheus_target_scrapes_sample_duplicate_timestamp_total",
		"prometheus_target_scrapes_sample_out_of_bounds_total",
		"prometheus_target_scrapes_sample_out_of_order_total",
		"prometheus_target_sync_length_seconds",
		"prometheus_target_sync_length_seconds_count",
		"prometheus_target_sync_length_seconds_sum",
		"prometheus_template_text_expansion_failures_total",
		"prometheus_template_text_expansions_total",
		"prometheus_treecache_watcher_goroutines",
		"prometheus_treecache_zookeeper_failures_total",
		"prometheus_tsdb_blocks_loaded",
		"prometheus_tsdb_checkpoint_creations_failed_total",
		"prometheus_tsdb_checkpoint_creations_total",
		"prometheus_tsdb_checkpoint_deletions_failed_total",
		"prometheus_tsdb_checkpoint_deletions_total",
		"prometheus_tsdb_compaction_chunk_range_seconds_bucket",
		"prometheus_tsdb_compaction_chunk_range_seconds_count",
		"prometheus_tsdb_compaction_chunk_range_seconds_sum",
		"prometheus_tsdb_compaction_chunk_samples_bucket",
		"prometheus_tsdb_compaction_chunk_samples_count",
		"prometheus_tsdb_compaction_chunk_samples_sum",
		"prometheus_tsdb_compaction_chunk_size_bytes_bucket",
		"prometheus_tsdb_compaction_chunk_size_bytes_count",
		"prometheus_tsdb_compaction_chunk_size_bytes_sum",
		"prometheus_tsdb_compaction_duration_seconds_bucket",
		"prometheus_tsdb_compaction_duration_seconds_count",
		"prometheus_tsdb_compaction_duration_seconds_sum",
		"prometheus_tsdb_compaction_populating_block",
		"prometheus_tsdb_compactions_failed_total",
		"prometheus_tsdb_compactions_skipped_total",
		"prometheus_tsdb_compactions_total",
		"prometheus_tsdb_compactions_triggered_total",
		"prometheus_tsdb_data_replay_duration_seconds",
		"prometheus_tsdb_head_active_appenders",
		"prometheus_tsdb_head_chunks",
		"prometheus_tsdb_head_chunks_created_total",
		"prometheus_tsdb_head_chunks_removed_total",
		"prometheus_tsdb_head_gc_duration_seconds_count",
		"prometheus_tsdb_head_gc_duration_seconds_sum",
		"prometheus_tsdb_head_max_time",
		"prometheus_tsdb_head_max_time_seconds",
		"prometheus_tsdb_head_min_time",
		"prometheus_tsdb_head_min_time_seconds",
		"prometheus_tsdb_head_samples_appended_total",
		"prometheus_tsdb_head_series",
		"prometheus_tsdb_head_series_created_total",
		"prometheus_tsdb_head_series_not_found_total",
		"prometheus_tsdb_head_series_removed_total",
		"prometheus_tsdb_head_truncations_failed_total",
		"prometheus_tsdb_head_truncations_total",
		"prometheus_tsdb_isolation_high_watermark",
		"prometheus_tsdb_isolation_low_watermark",
		"prometheus_tsdb_lowest_timestamp",
		"prometheus_tsdb_lowest_timestamp_seconds",
		"prometheus_tsdb_mmap_chunk_corruptions_total",
		"prometheus_tsdb_out_of_bound_samples_total",
		"prometheus_tsdb_out_of_order_samples_total",
		"prometheus_tsdb_reloads_failures_total",
		"prometheus_tsdb_reloads_total",
		"prometheus_tsdb_retention_limit_bytes",
		"prometheus_tsdb_size_retentions_total",
		"prometheus_tsdb_storage_blocks_bytes",
		"prometheus_tsdb_symbol_table_size_bytes",
		"prometheus_tsdb_time_retentions_total",
		"prometheus_tsdb_tombstone_cleanup_seconds_bucket",
		"prometheus_tsdb_tombstone_cleanup_seconds_count",
		"prometheus_tsdb_tombstone_cleanup_seconds_sum",
		"prometheus_tsdb_vertical_compactions_total",
		"prometheus_tsdb_wal_completed_pages_total",
		"prometheus_tsdb_wal_corruptions_total",
		"prometheus_tsdb_wal_fsync_duration_seconds",
		"prometheus_tsdb_wal_fsync_duration_seconds_count",
		"prometheus_tsdb_wal_fsync_duration_seconds_sum",
		"prometheus_tsdb_wal_page_flushes_total",
		"prometheus_tsdb_wal_segment_current",
		"prometheus_tsdb_wal_truncate_duration_seconds_count",
		"prometheus_tsdb_wal_truncate_duration_seconds_sum",
		"prometheus_tsdb_wal_truncations_failed_total",
		"prometheus_tsdb_wal_truncations_total",
		"prometheus_tsdb_wal_writes_failed_total",
		"prometheus_web_federation_errors_total",
		"prometheus_web_federation_warnings_total",
		"promhttp_metric_handler_requests_in_flight",
		"promhttp_metric_handler_requests_total",
		"raft_engine_allocate_log_duration_seconds_bucket",
		"raft_engine_allocate_log_duration_seconds_count",
		"raft_engine_allocate_log_duration_seconds_sum",
		"raft_engine_log_entry_count",
		"raft_engine_log_file_count",
		"raft_engine_memory_usage",
		"raft_engine_purge_duration_seconds_bucket",
		"raft_engine_purge_duration_seconds_count",
		"raft_engine_purge_duration_seconds_sum",
		"raft_engine_read_entry_count_bucket",
		"raft_engine_read_entry_count_count",
		"raft_engine_read_entry_count_sum",
		"raft_engine_read_entry_duration_seconds_bucket",
		"raft_engine_read_entry_duration_seconds_count",
		"raft_engine_read_entry_duration_seconds_sum",
		"raft_engine_read_message_duration_seconds_bucket",
		"raft_engine_read_message_duration_seconds_count",
		"raft_engine_read_message_duration_seconds_sum",
		"raft_engine_sync_log_duration_seconds_bucket",
		"raft_engine_sync_log_duration_seconds_count",
		"raft_engine_sync_log_duration_seconds_sum",
		"raft_engine_write_apply_duration_seconds_bucket",
		"raft_engine_write_apply_duration_seconds_count",
		"raft_engine_write_apply_duration_seconds_sum",
		"raft_engine_write_duration_seconds_bucket",
		"raft_engine_write_duration_seconds_count",
		"raft_engine_write_duration_seconds_sum",
		"raft_engine_write_leader_duration_seconds_bucket",
		"raft_engine_write_leader_duration_seconds_count",
		"raft_engine_write_leader_duration_seconds_sum",
		"raft_engine_write_preprocess_duration_seconds_bucket",
		"raft_engine_write_preprocess_duration_seconds_count",
		"raft_engine_write_preprocess_duration_seconds_sum",
		"raft_engine_write_size_bucket",
		"raft_engine_write_size_count",
		"raft_engine_write_size_sum",
		"scrape_duration_seconds",
		"scrape_samples_post_metric_relabeling",
		"scrape_samples_scraped",
		"scrape_series_added",
		"tidb_autoid_operation_duration_seconds_bucket",
		"tidb_autoid_operation_duration_seconds_count",
		"tidb_autoid_operation_duration_seconds_sum",
		"tidb_config_status",
		"tidb_ddl_deploy_syncer_duration_seconds_bucket",
		"tidb_ddl_deploy_syncer_duration_seconds_count",
		"tidb_ddl_deploy_syncer_duration_seconds_sum",
		"tidb_ddl_handle_job_duration_seconds_bucket",
		"tidb_ddl_handle_job_duration_seconds_count",
		"tidb_ddl_handle_job_duration_seconds_sum",
		"tidb_ddl_job_table_duration_seconds_bucket",
		"tidb_ddl_job_table_duration_seconds_count",
		"tidb_ddl_job_table_duration_seconds_sum",
		"tidb_ddl_owner_handle_syncer_duration_seconds_bucket",
		"tidb_ddl_owner_handle_syncer_duration_seconds_count",
		"tidb_ddl_owner_handle_syncer_duration_seconds_sum",
		"tidb_ddl_running_job_count",
		"tidb_ddl_update_self_ver_duration_seconds_bucket",
		"tidb_ddl_update_self_ver_duration_seconds_count",
		"tidb_ddl_update_self_ver_duration_seconds_sum",
		"tidb_ddl_waiting_jobs",
		"tidb_ddl_worker_operation_duration_seconds_bucket",
		"tidb_ddl_worker_operation_duration_seconds_count",
		"tidb_ddl_worker_operation_duration_seconds_sum",
		"tidb_ddl_worker_operation_total",
		"tidb_distsql_copr_cache",
		"tidb_distsql_copr_closest_read",
		"tidb_distsql_copr_resp_size_bucket",
		"tidb_distsql_copr_resp_size_count",
		"tidb_distsql_copr_resp_size_sum",
		"tidb_distsql_handle_query_duration_seconds_bucket",
		"tidb_distsql_handle_query_duration_seconds_count",
		"tidb_distsql_handle_query_duration_seconds_sum",
		"tidb_distsql_partial_num_bucket",
		"tidb_distsql_partial_num_count",
		"tidb_distsql_partial_num_sum",
		"tidb_distsql_scan_keys_num_bucket",
		"tidb_distsql_scan_keys_num_count",
		"tidb_distsql_scan_keys_num_sum",
		"tidb_distsql_scan_keys_partial_num_bucket",
		"tidb_distsql_scan_keys_partial_num_count",
		"tidb_distsql_scan_keys_partial_num_sum",
		"tidb_domain_infocache_counters",
		"tidb_domain_load_privilege_total",
		"tidb_domain_load_schema_duration_seconds_bucket",
		"tidb_domain_load_schema_duration_seconds_count",
		"tidb_domain_load_schema_duration_seconds_sum",
		"tidb_domain_load_schema_total",
		"tidb_domain_load_sysvarcache_total",
		"tidb_executor_expensive_total",
		"tidb_executor_phase_duration_seconds_count",
		"tidb_executor_phase_duration_seconds_sum",
		"tidb_executor_statement_total",
		"tidb_log_backup_advancer_owner",
		"tidb_log_backup_advancer_tick_duration_sec_bucket",
		"tidb_log_backup_advancer_tick_duration_sec_count",
		"tidb_log_backup_advancer_tick_duration_sec_sum",
		"tidb_meta_autoid_duration_seconds_bucket",
		"tidb_meta_autoid_duration_seconds_count",
		"tidb_meta_autoid_duration_seconds_sum",
		"tidb_meta_operation_duration_seconds_bucket",
		"tidb_meta_operation_duration_seconds_count",
		"tidb_meta_operation_duration_seconds_sum",
		"tidb_monitor_keep_alive_total",
		"tidb_monitor_time_jump_back_total",
		"tidb_owner_campaign_owner_total",
		"tidb_owner_new_session_duration_seconds_bucket",
		"tidb_owner_new_session_duration_seconds_count",
		"tidb_owner_new_session_duration_seconds_sum",
		"tidb_owner_watch_owner_total",
		"tidb_server_affected_rows",
		"tidb_server_conn_idle_duration_seconds_bucket",
		"tidb_server_conn_idle_duration_seconds_count",
		"tidb_server_conn_idle_duration_seconds_sum",
		"tidb_server_connections",
		"tidb_server_cpu_profile_total",
		"tidb_server_critical_error_total",
		"tidb_server_disconnection_total",
		"tidb_server_event_total",
		"tidb_server_get_token_duration_seconds_bucket",
		"tidb_server_get_token_duration_seconds_count",
		"tidb_server_get_token_duration_seconds_sum",
		"tidb_server_gogc",
		"tidb_server_handle_query_duration_seconds_bucket",
		"tidb_server_handle_query_duration_seconds_count",
		"tidb_server_handle_query_duration_seconds_sum",
		"tidb_server_handshake_error_total",
		"tidb_server_info",
		"tidb_server_load_table_cache_seconds_bucket",
		"tidb_server_load_table_cache_seconds_count",
		"tidb_server_load_table_cache_seconds_sum",
		"tidb_server_maxprocs",
		"tidb_server_multi_query_num_bucket",
		"tidb_server_multi_query_num_count",
		"tidb_server_multi_query_num_sum",
		"tidb_server_packet_io_bytes",
		"tidb_server_pd_api_execution_duration_seconds_bucket",
		"tidb_server_pd_api_execution_duration_seconds_count",
		"tidb_server_pd_api_execution_duration_seconds_sum",
		"tidb_server_pd_api_request_total",
		"tidb_server_plan_cache_miss_total",
		"tidb_server_plan_cache_total",
		"tidb_server_prepared_stmts",
		"tidb_server_query_total",
		"tidb_server_rc_check_ts_conflict_total",
		"tidb_server_read_from_tablecache_total",
		"tidb_server_slow_query_cop_duration_seconds_bucket",
		"tidb_server_slow_query_cop_duration_seconds_count",
		"tidb_server_slow_query_cop_duration_seconds_sum",
		"tidb_server_slow_query_process_duration_seconds_bucket",
		"tidb_server_slow_query_process_duration_seconds_count",
		"tidb_server_slow_query_process_duration_seconds_sum",
		"tidb_server_slow_query_wait_duration_seconds_bucket",
		"tidb_server_slow_query_wait_duration_seconds_count",
		"tidb_server_slow_query_wait_duration_seconds_sum",
		"tidb_server_tiflash_failed_store",
		"tidb_server_tiflash_query_total",
		"tidb_server_tokens",
		"tidb_server_ttl_job_status",
		"tidb_server_ttl_phase_time",
		"tidb_server_ttl_processed_expired_rows",
		"tidb_server_ttl_query_duration_bucket",
		"tidb_server_ttl_query_duration_count",
		"tidb_server_ttl_query_duration_sum",
		"tidb_session_compile_duration_seconds_bucket",
		"tidb_session_compile_duration_seconds_count",
		"tidb_session_compile_duration_seconds_sum",
		"tidb_session_execute_duration_seconds_bucket",
		"tidb_session_execute_duration_seconds_count",
		"tidb_session_execute_duration_seconds_sum",
		"tidb_session_non_transactional_dml_count",
		"tidb_session_parse_duration_seconds_bucket",
		"tidb_session_parse_duration_seconds_count",
		"tidb_session_parse_duration_seconds_sum",
		"tidb_session_restricted_sql_total",
		"tidb_session_retry_num_bucket",
		"tidb_session_retry_num_count",
		"tidb_session_retry_num_sum",
		"tidb_session_statement_deadlock_detect_duration_seconds_bucket",
		"tidb_session_statement_deadlock_detect_duration_seconds_count",
		"tidb_session_statement_deadlock_detect_duration_seconds_sum",
		"tidb_session_statement_lock_keys_count_bucket",
		"tidb_session_statement_lock_keys_count_count",
		"tidb_session_statement_lock_keys_count_sum",
		"tidb_session_statement_pessimistic_retry_count_bucket",
		"tidb_session_statement_pessimistic_retry_count_count",
		"tidb_session_statement_pessimistic_retry_count_sum",
		"tidb_session_transaction_duration_seconds_bucket",
		"tidb_session_transaction_duration_seconds_count",
		"tidb_session_transaction_duration_seconds_sum",
		"tidb_session_transaction_statement_num_bucket",
		"tidb_session_transaction_statement_num_count",
		"tidb_session_transaction_statement_num_sum",
		"tidb_session_txn_state_entering_count",
		"tidb_session_txn_state_seconds_bucket",
		"tidb_session_txn_state_seconds_count",
		"tidb_session_txn_state_seconds_sum",
		"tidb_session_validate_read_ts_from_pd_count",
		"tidb_sli_small_txn_write_duration_seconds_bucket",
		"tidb_sli_small_txn_write_duration_seconds_count",
		"tidb_sli_small_txn_write_duration_seconds_sum",
		"tidb_sli_tikv_read_throughput_bucket",
		"tidb_sli_tikv_read_throughput_count",
		"tidb_sli_tikv_read_throughput_sum",
		"tidb_sli_tikv_small_read_duration_bucket",
		"tidb_sli_tikv_small_read_duration_count",
		"tidb_sli_tikv_small_read_duration_sum",
		"tidb_sli_txn_write_throughput_bucket",
		"tidb_sli_txn_write_throughput_count",
		"tidb_sli_txn_write_throughput_sum",
		"tidb_statistics_auto_analyze_duration_seconds_bucket",
		"tidb_statistics_auto_analyze_duration_seconds_count",
		"tidb_statistics_auto_analyze_duration_seconds_sum",
		"tidb_statistics_fast_analyze_status_bucket",
		"tidb_statistics_fast_analyze_status_count",
		"tidb_statistics_fast_analyze_status_sum",
		"tidb_statistics_high_error_rate_feedback_total",
		"tidb_statistics_pseudo_estimation_total",
		"tidb_statistics_read_stats_latency_millis_bucket",
		"tidb_statistics_read_stats_latency_millis_count",
		"tidb_statistics_read_stats_latency_millis_sum",
		"tidb_statistics_stats_cache_lru_op",
		"tidb_statistics_stats_cache_lru_val",
		"tidb_statistics_stats_healthy",
		"tidb_statistics_stats_inaccuracy_rate_bucket",
		"tidb_statistics_stats_inaccuracy_rate_count",
		"tidb_statistics_stats_inaccuracy_rate_sum",
		"tidb_statistics_sync_load_latency_millis_bucket",
		"tidb_statistics_sync_load_latency_millis_count",
		"tidb_statistics_sync_load_latency_millis_sum",
		"tidb_statistics_sync_load_timeout_total",
		"tidb_statistics_sync_load_total",
		"tidb_tikvclient_async_commit_txn_counter",
		"tidb_tikvclient_backoff_seconds_bucket",
		"tidb_tikvclient_backoff_seconds_count",
		"tidb_tikvclient_backoff_seconds_sum",
		"tidb_tikvclient_batch_client_no_available_connection_total",
		"tidb_tikvclient_batch_client_reset_bucket",
		"tidb_tikvclient_batch_client_reset_count",
		"tidb_tikvclient_batch_client_reset_sum",
		"tidb_tikvclient_batch_client_unavailable_seconds_bucket",
		"tidb_tikvclient_batch_client_unavailable_seconds_count",
		"tidb_tikvclient_batch_client_unavailable_seconds_sum",
		"tidb_tikvclient_batch_client_wait_connection_establish_bucket",
		"tidb_tikvclient_batch_client_wait_connection_establish_count",
		"tidb_tikvclient_batch_client_wait_connection_establish_sum",
		"tidb_tikvclient_batch_executor_token_wait_duration_bucket",
		"tidb_tikvclient_batch_executor_token_wait_duration_count",
		"tidb_tikvclient_batch_executor_token_wait_duration_sum",
		"tidb_tikvclient_batch_pending_requests_bucket",
		"tidb_tikvclient_batch_pending_requests_count",
		"tidb_tikvclient_batch_pending_requests_sum",
		"tidb_tikvclient_batch_recv_latency_bucket",
		"tidb_tikvclient_batch_recv_latency_count",
		"tidb_tikvclient_batch_recv_latency_sum",
		"tidb_tikvclient_batch_requests_bucket",
		"tidb_tikvclient_batch_requests_count",
		"tidb_tikvclient_batch_requests_sum",
		"tidb_tikvclient_batch_send_latency_bucket",
		"tidb_tikvclient_batch_send_latency_count",
		"tidb_tikvclient_batch_send_latency_sum",
		"tidb_tikvclient_batch_wait_duration_bucket",
		"tidb_tikvclient_batch_wait_duration_count",
		"tidb_tikvclient_batch_wait_duration_sum",
		"tidb_tikvclient_batch_wait_overload",
		"tidb_tikvclient_commit_txn_counter",
		"tidb_tikvclient_cop_duration_seconds_bucket",
		"tidb_tikvclient_cop_duration_seconds_count",
		"tidb_tikvclient_cop_duration_seconds_sum",
		"tidb_tikvclient_gc_config",
		"tidb_tikvclient_gc_region_too_many_locks",
		"tidb_tikvclient_gc_seconds_bucket",
		"tidb_tikvclient_gc_seconds_count",
		"tidb_tikvclient_gc_seconds_sum",
		"tidb_tikvclient_gc_worker_actions_total",
		"tidb_tikvclient_kv_status_api_count",
		"tidb_tikvclient_load_region_cache_seconds_bucket",
		"tidb_tikvclient_load_region_cache_seconds_count",
		"tidb_tikvclient_load_region_cache_seconds_sum",
		"tidb_tikvclient_load_safepoint_total",
		"tidb_tikvclient_local_latch_wait_seconds_bucket",
		"tidb_tikvclient_local_latch_wait_seconds_count",
		"tidb_tikvclient_local_latch_wait_seconds_sum",
		"tidb_tikvclient_lock_cleanup_task_total",
		"tidb_tikvclient_lock_resolver_actions_total",
		"tidb_tikvclient_min_safets_gap_seconds",
		"tidb_tikvclient_one_pc_txn_counter",
		"tidb_tikvclient_pessimistic_lock_keys_duration_bucket",
		"tidb_tikvclient_pessimistic_lock_keys_duration_count",
		"tidb_tikvclient_pessimistic_lock_keys_duration_sum",
		"tidb_tikvclient_prewrite_assertion_count",
		"tidb_tikvclient_range_task_push_duration_bucket",
		"tidb_tikvclient_range_task_push_duration_count",
		"tidb_tikvclient_range_task_push_duration_sum",
		"tidb_tikvclient_range_task_stats",
		"tidb_tikvclient_rawkv_cmd_seconds_bucket",
		"tidb_tikvclient_rawkv_cmd_seconds_count",
		"tidb_tikvclient_rawkv_cmd_seconds_sum",
		"tidb_tikvclient_rawkv_kv_size_bytes_bucket",
		"tidb_tikvclient_rawkv_kv_size_bytes_count",
		"tidb_tikvclient_rawkv_kv_size_bytes_sum",
		"tidb_tikvclient_region_cache_operations_total",
		"tidb_tikvclient_region_err_total",
		"tidb_tikvclient_request_counter",
		"tidb_tikvclient_request_retry_times_bucket",
		"tidb_tikvclient_request_retry_times_count",
		"tidb_tikvclient_request_retry_times_sum",
		"tidb_tikvclient_request_seconds_bucket",
		"tidb_tikvclient_request_seconds_count",
		"tidb_tikvclient_request_seconds_sum",
		"tidb_tikvclient_request_time_counter",
		"tidb_tikvclient_rpc_net_latency_seconds_bucket",
		"tidb_tikvclient_rpc_net_latency_seconds_count",
		"tidb_tikvclient_rpc_net_latency_seconds_sum",
		"tidb_tikvclient_safets_update_counter",
		"tidb_tikvclient_ts_future_wait_seconds_bucket",
		"tidb_tikvclient_ts_future_wait_seconds_count",
		"tidb_tikvclient_ts_future_wait_seconds_sum",
		"tidb_tikvclient_ttl_lifetime_reach_total",
		"tidb_tikvclient_txn_cmd_duration_seconds_bucket",
		"tidb_tikvclient_txn_cmd_duration_seconds_count",
		"tidb_tikvclient_txn_cmd_duration_seconds_sum",
		"tidb_tikvclient_txn_commit_backoff_count_bucket",
		"tidb_tikvclient_txn_commit_backoff_count_count",
		"tidb_tikvclient_txn_commit_backoff_count_sum",
		"tidb_tikvclient_txn_commit_backoff_seconds_bucket",
		"tidb_tikvclient_txn_commit_backoff_seconds_count",
		"tidb_tikvclient_txn_commit_backoff_seconds_sum",
		"tidb_tikvclient_txn_heart_beat_bucket",
		"tidb_tikvclient_txn_heart_beat_count",
		"tidb_tikvclient_txn_heart_beat_sum",
		"tidb_tikvclient_txn_regions_num_bucket",
		"tidb_tikvclient_txn_regions_num_count",
		"tidb_tikvclient_txn_regions_num_sum",
		"tidb_tikvclient_txn_write_kv_num_bucket",
		"tidb_tikvclient_txn_write_kv_num_count",
		"tidb_tikvclient_txn_write_kv_num_sum",
		"tidb_tikvclient_txn_write_size_bytes_bucket",
		"tidb_tikvclient_txn_write_size_bytes_count",
		"tidb_tikvclient_txn_write_size_bytes_sum",
		"tidb_topsql_ignored_total",
		"tidb_topsql_report_data_total_bucket",
		"tidb_topsql_report_data_total_count",
		"tidb_topsql_report_data_total_sum",
		"tidb_topsql_report_duration_seconds_bucket",
		"tidb_topsql_report_duration_seconds_count",
		"tidb_topsql_report_duration_seconds_sum",
		"tiflash_coprocessor_executor_count",
		"tiflash_coprocessor_handling_request_count",
		"tiflash_coprocessor_request_count",
		"tiflash_coprocessor_request_duration_seconds_bucket",
		"tiflash_coprocessor_request_duration_seconds_count",
		"tiflash_coprocessor_request_duration_seconds_sum",
		"tiflash_coprocessor_request_error",
		"tiflash_coprocessor_request_handle_seconds_bucket",
		"tiflash_coprocessor_request_handle_seconds_count",
		"tiflash_coprocessor_request_handle_seconds_sum",
		"tiflash_coprocessor_request_memory_usage_bucket",
		"tiflash_coprocessor_request_memory_usage_count",
		"tiflash_coprocessor_request_memory_usage_sum",
		"tiflash_coprocessor_response_bytes",
		"tiflash_mpp_task_manager",
		"tiflash_object_count",
		"tiflash_proxy_process_cpu_seconds_total",
		"tiflash_proxy_process_resident_memory_bytes",
		"tiflash_proxy_process_start_time_seconds",
		"tiflash_proxy_process_virtual_memory_bytes",
		"tiflash_proxy_raft_engine_allocate_log_duration_seconds_bucket",
		"tiflash_proxy_raft_engine_allocate_log_duration_seconds_count",
		"tiflash_proxy_raft_engine_allocate_log_duration_seconds_sum",
		"tiflash_proxy_raft_engine_log_entry_count",
		"tiflash_proxy_raft_engine_log_file_count",
		"tiflash_proxy_raft_engine_memory_usage",
		"tiflash_proxy_raft_engine_purge_duration_seconds_bucket",
		"tiflash_proxy_raft_engine_purge_duration_seconds_count",
		"tiflash_proxy_raft_engine_purge_duration_seconds_sum",
		"tiflash_proxy_thread_cpu_seconds_total",
		"tiflash_proxy_threads_io_bytes_total",
		"tiflash_proxy_threads_state",
		"tiflash_proxy_tikv_config_raftstore",
		"tiflash_proxy_tikv_config_rocksdb",
		"tiflash_proxy_tikv_engine_blob_cache_size_bytes",
		"tiflash_proxy_tikv_engine_block_cache_size_bytes",
		"tiflash_proxy_tikv_engine_bloom_efficiency",
		"tiflash_proxy_tikv_engine_bytes_compressed",
		"tiflash_proxy_tikv_engine_bytes_decompressed",
		"tiflash_proxy_tikv_engine_bytes_per_read",
		"tiflash_proxy_tikv_engine_bytes_per_write",
		"tiflash_proxy_tikv_engine_cache_efficiency",
		"tiflash_proxy_tikv_engine_compaction_flow_bytes",
		"tiflash_proxy_tikv_engine_compaction_key_drop",
		"tiflash_proxy_tikv_engine_compaction_outfile_sync_micro_seconds",
		"tiflash_proxy_tikv_engine_compaction_time",
		"tiflash_proxy_tikv_engine_compression_time_nanos",
		"tiflash_proxy_tikv_engine_decompression_time_nanos",
		"tiflash_proxy_tikv_engine_estimate_num_keys",
		"tiflash_proxy_tikv_engine_file_status",
		"tiflash_proxy_tikv_engine_flow_bytes",
		"tiflash_proxy_tikv_engine_get_micro_seconds",
		"tiflash_proxy_tikv_engine_get_served",
		"tiflash_proxy_tikv_engine_hard_rate_limit_delay_count",
		"tiflash_proxy_tikv_engine_locate",
		"tiflash_proxy_tikv_engine_manifest_file_sync_micro_seconds",
		"tiflash_proxy_tikv_engine_memory_bytes",
		"tiflash_proxy_tikv_engine_memtable_efficiency",
		"tiflash_proxy_tikv_engine_num_files_at_level",
		"tiflash_proxy_tikv_engine_num_files_in_single_compaction",
		"tiflash_proxy_tikv_engine_num_immutable_mem_table",
		"tiflash_proxy_tikv_engine_num_snapshots",
		"tiflash_proxy_tikv_engine_num_subcompaction_scheduled",
		"tiflash_proxy_tikv_engine_oldest_snapshot_duration",
		"tiflash_proxy_tikv_engine_pending_compaction_bytes",
		"tiflash_proxy_tikv_engine_read_amp_flow_bytes",
		"tiflash_proxy_tikv_engine_seek_micro_seconds",
		"tiflash_proxy_tikv_engine_size_bytes",
		"tiflash_proxy_tikv_engine_soft_rate_limit_delay_count",
		"tiflash_proxy_tikv_engine_sst_read_micros",
		"tiflash_proxy_tikv_engine_stall_l0_num_files_count",
		"tiflash_proxy_tikv_engine_stall_l0_slowdown_count",
		"tiflash_proxy_tikv_engine_stall_memtable_compaction_count",
		"tiflash_proxy_tikv_engine_stall_micro_seconds",
		"tiflash_proxy_tikv_engine_table_sync_micro_seconds",
		"tiflash_proxy_tikv_engine_wal_file_sync_micro_seconds",
		"tiflash_proxy_tikv_engine_wal_file_synced",
		"tiflash_proxy_tikv_engine_write_micro_seconds",
		"tiflash_proxy_tikv_engine_write_served",
		"tiflash_proxy_tikv_engine_write_stall",
		"tiflash_proxy_tikv_engine_write_stall_reason",
		"tiflash_proxy_tikv_engine_write_wal_time_micro_seconds",
		"tiflash_proxy_tikv_futurepool_handled_task_total",
		"tiflash_proxy_tikv_futurepool_pending_task_total",
		"tiflash_proxy_tikv_io_bytes",
		"tiflash_proxy_tikv_load_base_split_duration_seconds_bucket",
		"tiflash_proxy_tikv_load_base_split_duration_seconds_count",
		"tiflash_proxy_tikv_load_base_split_duration_seconds_sum",
		"tiflash_proxy_tikv_pd_pending_tso_request_total",
		"tiflash_proxy_tikv_pd_reconnect_total",
		"tiflash_proxy_tikv_pd_request_duration_seconds_bucket",
		"tiflash_proxy_tikv_pd_request_duration_seconds_count",
		"tiflash_proxy_tikv_pd_request_duration_seconds_sum",
		"tiflash_proxy_tikv_pending_delete_ranges_of_stale_peer",
		"tiflash_proxy_tikv_raftstore_apply_duration_secs_bucket",
		"tiflash_proxy_tikv_raftstore_apply_duration_secs_count",
		"tiflash_proxy_tikv_raftstore_apply_duration_secs_sum",
		"tiflash_proxy_tikv_raftstore_apply_wait_time_duration_secs_bucket",
		"tiflash_proxy_tikv_raftstore_apply_wait_time_duration_secs_count",
		"tiflash_proxy_tikv_raftstore_apply_wait_time_duration_secs_sum",
		"tiflash_proxy_tikv_raftstore_commit_log_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_commit_log_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_commit_log_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_event_duration_bucket",
		"tiflash_proxy_tikv_raftstore_event_duration_count",
		"tiflash_proxy_tikv_raftstore_event_duration_sum",
		"tiflash_proxy_tikv_raftstore_inspect_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_inspect_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_inspect_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_leader_missing",
		"tiflash_proxy_tikv_raftstore_peer_msg_len_bucket",
		"tiflash_proxy_tikv_raftstore_peer_msg_len_count",
		"tiflash_proxy_tikv_raftstore_peer_msg_len_sum",
		"tiflash_proxy_tikv_raftstore_proposal_total",
		"tiflash_proxy_tikv_raftstore_propose_log_size_bucket",
		"tiflash_proxy_tikv_raftstore_propose_log_size_count",
		"tiflash_proxy_tikv_raftstore_propose_log_size_sum",
		"tiflash_proxy_tikv_raftstore_raft_dropped_message_total",
		"tiflash_proxy_tikv_raftstore_raft_invalid_proposal_total",
		"tiflash_proxy_tikv_raftstore_raft_log_gc_skipped",
		"tiflash_proxy_tikv_raftstore_raft_process_duration_secs_bucket",
		"tiflash_proxy_tikv_raftstore_raft_process_duration_secs_count",
		"tiflash_proxy_tikv_raftstore_raft_process_duration_secs_sum",
		"tiflash_proxy_tikv_raftstore_raft_ready_handled_total",
		"tiflash_proxy_tikv_raftstore_raft_sent_message_total",
		"tiflash_proxy_tikv_raftstore_region_count",
		"tiflash_proxy_tikv_raftstore_request_wait_time_duration_secs_bucket",
		"tiflash_proxy_tikv_raftstore_request_wait_time_duration_secs_count",
		"tiflash_proxy_tikv_raftstore_request_wait_time_duration_secs_sum",
		"tiflash_proxy_tikv_raftstore_slow_score",
		"tiflash_proxy_tikv_raftstore_snapshot_traffic_total",
		"tiflash_proxy_tikv_raftstore_store_duration_secs_bucket",
		"tiflash_proxy_tikv_raftstore_store_duration_secs_count",
		"tiflash_proxy_tikv_raftstore_store_duration_secs_sum",
		"tiflash_proxy_tikv_raftstore_store_wf_batch_wait_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_store_wf_batch_wait_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_store_wf_batch_wait_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_store_wf_before_write_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_store_wf_before_write_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_store_wf_before_write_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_store_wf_commit_log_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_store_wf_commit_log_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_store_wf_commit_log_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_store_wf_commit_not_persist_log_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_store_wf_commit_not_persist_log_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_store_wf_commit_not_persist_log_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_store_wf_persist_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_store_wf_persist_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_store_wf_persist_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_store_wf_send_proposal_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_store_wf_send_proposal_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_store_wf_send_proposal_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_store_wf_send_to_queue_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_store_wf_send_to_queue_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_store_wf_send_to_queue_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_store_wf_write_end_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_store_wf_write_end_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_store_wf_write_end_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_store_wf_write_kvdb_end_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_store_wf_write_kvdb_end_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_store_wf_write_kvdb_end_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_store_write_msg_block_wait_duration_seconds_bucket",
		"tiflash_proxy_tikv_raftstore_store_write_msg_block_wait_duration_seconds_count",
		"tiflash_proxy_tikv_raftstore_store_write_msg_block_wait_duration_seconds_sum",
		"tiflash_proxy_tikv_raftstore_store_write_task_wait_duration_secs_bucket",
		"tiflash_proxy_tikv_raftstore_store_write_task_wait_duration_secs_count",
		"tiflash_proxy_tikv_raftstore_store_write_task_wait_duration_secs_sum",
		"tiflash_proxy_tikv_rate_limiter_max_bytes_per_sec",
		"tiflash_proxy_tikv_read_qps_topn",
		"tiflash_proxy_tikv_region_read_bytes_bucket",
		"tiflash_proxy_tikv_region_read_bytes_count",
		"tiflash_proxy_tikv_region_read_bytes_sum",
		"tiflash_proxy_tikv_region_read_keys_bucket",
		"tiflash_proxy_tikv_region_read_keys_count",
		"tiflash_proxy_tikv_region_read_keys_sum",
		"tiflash_proxy_tikv_region_written_bytes_bucket",
		"tiflash_proxy_tikv_region_written_bytes_count",
		"tiflash_proxy_tikv_region_written_bytes_sum",
		"tiflash_proxy_tikv_region_written_keys_bucket",
		"tiflash_proxy_tikv_region_written_keys_count",
		"tiflash_proxy_tikv_region_written_keys_sum",
		"tiflash_proxy_tikv_scheduler_throttle_cf",
		"tiflash_proxy_tikv_scheduler_write_flow",
		"tiflash_proxy_tikv_server_cpu_cores_quota",
		"tiflash_proxy_tikv_server_info",
		"tiflash_proxy_tikv_server_mem_trace_sum",
		"tiflash_proxy_tikv_server_memory_usage",
		"tiflash_proxy_tikv_store_size_bytes",
		"tiflash_proxy_tikv_unified_read_pool_running_tasks",
		"tiflash_proxy_tikv_unified_read_pool_thread_count",
		"tiflash_proxy_tikv_worker_pending_task_total",
		"tiflash_proxy_tikv_yatp_pool_schedule_wait_duration_bucket",
		"tiflash_proxy_tikv_yatp_pool_schedule_wait_duration_count",
		"tiflash_proxy_tikv_yatp_pool_schedule_wait_duration_sum",
		"tiflash_raft_apply_write_command_duration_seconds_bucket",
		"tiflash_raft_apply_write_command_duration_seconds_count",
		"tiflash_raft_apply_write_command_duration_seconds_sum",
		"tiflash_raft_command_duration_seconds_bucket",
		"tiflash_raft_command_duration_seconds_count",
		"tiflash_raft_command_duration_seconds_sum",
		"tiflash_raft_process_keys",
		"tiflash_raft_read_index_count",
		"tiflash_raft_read_index_duration_seconds_bucket",
		"tiflash_raft_read_index_duration_seconds_count",
		"tiflash_raft_read_index_duration_seconds_sum",
		"tiflash_raft_upstream_latency_bucket",
		"tiflash_raft_upstream_latency_count",
		"tiflash_raft_upstream_latency_sum",
		"tiflash_raft_wait_index_duration_seconds_bucket",
		"tiflash_raft_wait_index_duration_seconds_count",
		"tiflash_raft_wait_index_duration_seconds_sum",
		"tiflash_raft_write_data_to_storage_duration_seconds_bucket",
		"tiflash_raft_write_data_to_storage_duration_seconds_count",
		"tiflash_raft_write_data_to_storage_duration_seconds_sum",
		"tiflash_schema_apply_count",
		"tiflash_schema_apply_duration_seconds_bucket",
		"tiflash_schema_apply_duration_seconds_count",
		"tiflash_schema_apply_duration_seconds_sum",
		"tiflash_schema_applying",
		"tiflash_schema_internal_ddl_count",
		"tiflash_schema_trigger_count",
		"tiflash_schema_version",
		"tiflash_server_info",
		"tiflash_storage_command_count",
		"tiflash_storage_io_limiter",
		"tiflash_storage_logical_throughput_bytes_bucket",
		"tiflash_storage_logical_throughput_bytes_count",
		"tiflash_storage_logical_throughput_bytes_sum",
		"tiflash_storage_page_gc_count",
		"tiflash_storage_page_gc_duration_seconds_bucket",
		"tiflash_storage_page_gc_duration_seconds_count",
		"tiflash_storage_page_gc_duration_seconds_sum",
		"tiflash_storage_page_write_batch_size_bucket",
		"tiflash_storage_page_write_batch_size_count",
		"tiflash_storage_page_write_batch_size_sum",
		"tiflash_storage_page_write_duration_seconds_bucket",
		"tiflash_storage_page_write_duration_seconds_count",
		"tiflash_storage_page_write_duration_seconds_sum",
		"tiflash_storage_read_tasks_count",
		"tiflash_storage_read_thread_counter",
		"tiflash_storage_read_thread_gauge",
		"tiflash_storage_read_thread_seconds_bucket",
		"tiflash_storage_read_thread_seconds_count",
		"tiflash_storage_read_thread_seconds_sum",
		"tiflash_storage_rough_set_filter_rate_bucket",
		"tiflash_storage_rough_set_filter_rate_count",
		"tiflash_storage_rough_set_filter_rate_sum",
		"tiflash_storage_subtask_count",
		"tiflash_storage_subtask_duration_seconds_bucket",
		"tiflash_storage_subtask_duration_seconds_count",
		"tiflash_storage_subtask_duration_seconds_sum",
		"tiflash_storage_throughput_bytes",
		"tiflash_storage_throughput_rows",
		"tiflash_storage_write_stall_duration_seconds_bucket",
		"tiflash_storage_write_stall_duration_seconds_count",
		"tiflash_storage_write_stall_duration_seconds_sum",
		"tiflash_syncing_data_freshness_bucket",
		"tiflash_syncing_data_freshness_count",
		"tiflash_syncing_data_freshness_sum",
		"tiflash_system_asynchronous_metric_BlobDiskBytes",
		"tiflash_system_asynchronous_metric_BlobFileNums",
		"tiflash_system_asynchronous_metric_BlobValidBytes",
		"tiflash_system_asynchronous_metric_LogDiskBytes",
		"tiflash_system_asynchronous_metric_LogNums",
		"tiflash_system_asynchronous_metric_MarkCacheBytes",
		"tiflash_system_asynchronous_metric_MarkCacheFiles",
		"tiflash_system_asynchronous_metric_MaxDTBackgroundTasksLength",
		"tiflash_system_asynchronous_metric_MaxDTDeltaOldestSnapshotLifetime",
		"tiflash_system_asynchronous_metric_MaxDTMetaOldestSnapshotLifetime",
		"tiflash_system_asynchronous_metric_MaxDTStableOldestSnapshotLifetime",
		"tiflash_system_asynchronous_metric_PagesInMem",
		"tiflash_system_asynchronous_metric_Uptime",
		"tiflash_system_asynchronous_metric_jemalloc_active",
		"tiflash_system_asynchronous_metric_jemalloc_allocated",
		"tiflash_system_asynchronous_metric_jemalloc_background_thread_num_runs",
		"tiflash_system_asynchronous_metric_jemalloc_background_thread_num_threads",
		"tiflash_system_asynchronous_metric_jemalloc_background_thread_run_interval",
		"tiflash_system_asynchronous_metric_jemalloc_mapped",
		"tiflash_system_asynchronous_metric_jemalloc_metadata",
		"tiflash_system_asynchronous_metric_jemalloc_metadata_thp",
		"tiflash_system_asynchronous_metric_jemalloc_resident",
		"tiflash_system_asynchronous_metric_jemalloc_retained",
		"tiflash_system_asynchronous_metric_mmap_alive",
		"tiflash_system_current_metric_DT_DeltaCompact",
		"tiflash_system_current_metric_DT_DeltaFlush",
		"tiflash_system_current_metric_DT_DeltaIndexCacheSize",
		"tiflash_system_current_metric_DT_DeltaMerge",
		"tiflash_system_current_metric_DT_DeltaMergeTotalBytes",
		"tiflash_system_current_metric_DT_DeltaMergeTotalRows",
		"tiflash_system_current_metric_DT_PlaceIndexUpdate",
		"tiflash_system_current_metric_DT_SegmentMerge",
		"tiflash_system_current_metric_DT_SegmentReadTasks",
		"tiflash_system_current_metric_DT_SegmentSplit",
		"tiflash_system_current_metric_DT_SnapshotOfDeltaCompact",
		"tiflash_system_current_metric_DT_SnapshotOfDeltaMerge",
		"tiflash_system_current_metric_DT_SnapshotOfPlaceIndex",
		"tiflash_system_current_metric_DT_SnapshotOfRead",
		"tiflash_system_current_metric_DT_SnapshotOfReadRaw",
		"tiflash_system_current_metric_DT_SnapshotOfSegmentIngest",
		"tiflash_system_current_metric_DT_SnapshotOfSegmentMerge",
		"tiflash_system_current_metric_DT_SnapshotOfSegmentSplit",
		"tiflash_system_current_metric_GlobalStorageRunMode",
		"tiflash_system_current_metric_IOLimiterPendingBgReadReq",
		"tiflash_system_current_metric_IOLimiterPendingBgWriteReq",
		"tiflash_system_current_metric_IOLimiterPendingFgReadReq",
		"tiflash_system_current_metric_IOLimiterPendingFgWriteReq",
		"tiflash_system_current_metric_LogicalCPUCores",
		"tiflash_system_current_metric_MemoryCapacity",
		"tiflash_system_current_metric_MemoryTracking",
		"tiflash_system_current_metric_MemoryTrackingInBackgroundProcessingPool",
		"tiflash_system_current_metric_OpenFileForRead",
		"tiflash_system_current_metric_OpenFileForReadWrite",
		"tiflash_system_current_metric_OpenFileForWrite",
		"tiflash_system_current_metric_PSMVCCNumBase",
		"tiflash_system_current_metric_PSMVCCNumDelta",
		"tiflash_system_current_metric_PSMVCCNumSnapshots",
		"tiflash_system_current_metric_PSMVCCSnapshotsList",
		"tiflash_system_current_metric_RWLockActiveReaders",
		"tiflash_system_current_metric_RWLockActiveWriters",
		"tiflash_system_current_metric_RWLockWaitingReaders",
		"tiflash_system_current_metric_RWLockWaitingWriters",
		"tiflash_system_current_metric_RaftNumSnapshotsPendingApply",
		"tiflash_system_current_metric_RateLimiterPendingWriteRequest",
		"tiflash_system_current_metric_RegionPersisterRunMode",
		"tiflash_system_current_metric_StoragePoolMixMode",
		"tiflash_system_current_metric_StoragePoolV2Only",
		"tiflash_system_current_metric_StoragePoolV3Only",
		"tiflash_system_current_metric_StoreSizeAvailable",
		"tiflash_system_current_metric_StoreSizeCapacity",
		"tiflash_system_current_metric_StoreSizeUsed",
		"tiflash_system_profile_event_ChecksumDigestBytes",
		"tiflash_system_profile_event_ContextLock",
		"tiflash_system_profile_event_DMAppendDeltaCleanUp",
		"tiflash_system_profile_event_DMAppendDeltaCleanUpNS",
		"tiflash_system_profile_event_DMAppendDeltaCommitDisk",
		"tiflash_system_profile_event_DMAppendDeltaCommitDiskNS",
		"tiflash_system_profile_event_DMAppendDeltaCommitMemory",
		"tiflash_system_profile_event_DMAppendDeltaCommitMemoryNS",
		"tiflash_system_profile_event_DMAppendDeltaPrepare",
		"tiflash_system_profile_event_DMAppendDeltaPrepareNS",
		"tiflash_system_profile_event_DMCleanReadRows",
		"tiflash_system_profile_event_DMDeleteRange",
		"tiflash_system_profile_event_DMDeleteRangeNS",
		"tiflash_system_profile_event_DMDeltaMerge",
		"tiflash_system_profile_event_DMDeltaMergeNS",
		"tiflash_system_profile_event_DMFileFilterAftPKAndPackSet",
		"tiflash_system_profile_event_DMFileFilterAftRoughSet",
		"tiflash_system_profile_event_DMFileFilterNoFilter",
		"tiflash_system_profile_event_DMFlushDeltaCache",
		"tiflash_system_profile_event_DMFlushDeltaCacheNS",
		"tiflash_system_profile_event_DMPlace",
		"tiflash_system_profile_event_DMPlaceDeleteRange",
		"tiflash_system_profile_event_DMPlaceDeleteRangeNS",
		"tiflash_system_profile_event_DMPlaceNS",
		"tiflash_system_profile_event_DMPlaceUpsert",
		"tiflash_system_profile_event_DMPlaceUpsertNS",
		"tiflash_system_profile_event_DMSegmentGetSplitPoint",
		"tiflash_system_profile_event_DMSegmentGetSplitPointNS",
		"tiflash_system_profile_event_DMSegmentIngestDataByReplace",
		"tiflash_system_profile_event_DMSegmentIngestDataIntoDelta",
		"tiflash_system_profile_event_DMSegmentIsEmptyFastPath",
		"tiflash_system_profile_event_DMSegmentIsEmptySlowPath",
		"tiflash_system_profile_event_DMSegmentMerge",
		"tiflash_system_profile_event_DMSegmentMergeNS",
		"tiflash_system_profile_event_DMSegmentSplit",
		"tiflash_system_profile_event_DMSegmentSplitNS",
		"tiflash_system_profile_event_DMWriteBlock",
		"tiflash_system_profile_event_DMWriteBlockNS",
		"tiflash_system_profile_event_DMWriteFile",
		"tiflash_system_profile_event_DMWriteFileNS",
		"tiflash_system_profile_event_ExternalAggregationCompressedBytes",
		"tiflash_system_profile_event_ExternalAggregationUncompressedBytes",
		"tiflash_system_profile_event_FileFSync",
		"tiflash_system_profile_event_FileOpen",
		"tiflash_system_profile_event_FileOpenFailed",
		"tiflash_system_profile_event_MarkCacheHits",
		"tiflash_system_profile_event_MarkCacheMisses",
		"tiflash_system_profile_event_PSMBackgroundReadBytes",
		"tiflash_system_profile_event_PSMBackgroundWriteBytes",
		"tiflash_system_profile_event_PSMReadBytes",
		"tiflash_system_profile_event_PSMReadFailed",
		"tiflash_system_profile_event_PSMReadIOCalls",
		"tiflash_system_profile_event_PSMReadPages",
		"tiflash_system_profile_event_PSMVCCApplyOnCurrentBase",
		"tiflash_system_profile_event_PSMVCCApplyOnCurrentDelta",
		"tiflash_system_profile_event_PSMVCCApplyOnNewDelta",
		"tiflash_system_profile_event_PSMVCCCompactOnBase",
		"tiflash_system_profile_event_PSMVCCCompactOnBaseCommit",
		"tiflash_system_profile_event_PSMVCCCompactOnDelta",
		"tiflash_system_profile_event_PSMVCCCompactOnDeltaRebaseRejected",
		"tiflash_system_profile_event_PSMWriteBytes",
		"tiflash_system_profile_event_PSMWriteFailed",
		"tiflash_system_profile_event_PSMWriteIOCalls",
		"tiflash_system_profile_event_PSMWritePages",
		"tiflash_system_profile_event_PSV3MBlobExpansion",
		"tiflash_system_profile_event_PSV3MBlobReused",
		"tiflash_system_profile_event_Query",
		"tiflash_system_profile_event_RWLockAcquiredReadLocks",
		"tiflash_system_profile_event_RWLockAcquiredWriteLocks",
		"tiflash_system_profile_event_RWLockReadersWaitMilliseconds",
		"tiflash_system_profile_event_RWLockWritersWaitMilliseconds",
		"tiflash_system_profile_event_RaftWaitIndexTimeout",
		"tiflash_system_profile_event_ReadBufferAIORead",
		"tiflash_system_profile_event_ReadBufferAIOReadBytes",
		"tiflash_system_profile_event_ReadBufferFromFileDescriptorRead",
		"tiflash_system_profile_event_ReadBufferFromFileDescriptorReadBytes",
		"tiflash_system_profile_event_ReadBufferFromFileDescriptorReadFailed",
		"tiflash_system_profile_event_UncompressedCacheHits",
		"tiflash_system_profile_event_UncompressedCacheMisses",
		"tiflash_system_profile_event_UncompressedCacheWeightLost",
		"tiflash_system_profile_event_WriteBufferAIOWrite",
		"tiflash_system_profile_event_WriteBufferAIOWriteBytes",
		"tiflash_system_profile_event_WriteBufferFromFileDescriptorWrite",
		"tiflash_system_profile_event_WriteBufferFromFileDescriptorWriteBytes",
		"tiflash_task_scheduler",
		"tiflash_task_scheduler_waiting_duration_seconds_bucket",
		"tiflash_task_scheduler_waiting_duration_seconds_count",
		"tiflash_task_scheduler_waiting_duration_seconds_sum",
		"tiflash_thread_count",
		"tikv_allocator_stats",
		"tikv_backup_softlimit",
		"tikv_cdc_captured_region_total",
		"tikv_cdc_endpoint_pending_tasks",
		"tikv_cdc_old_value_cache_access",
		"tikv_cdc_old_value_cache_bytes",
		"tikv_cdc_old_value_cache_length",
		"tikv_cdc_old_value_cache_memory_quota",
		"tikv_cdc_old_value_cache_miss",
		"tikv_cdc_old_value_cache_miss_none",
		"tikv_cdc_region_resolve_status",
		"tikv_cdc_resolved_ts_advance_method",
		"tikv_cdc_sink_memory_bytes",
		"tikv_cdc_sink_memory_capacity",
		"tikv_config_raftstore",
		"tikv_config_rocksdb",
		"tikv_coprocessor_acquire_semaphore_type",
		"tikv_coprocessor_dag_request_count",
		"tikv_coprocessor_executor_count",
		"tikv_coprocessor_mem_lock_check_duration_seconds_bucket",
		"tikv_coprocessor_mem_lock_check_duration_seconds_count",
		"tikv_coprocessor_mem_lock_check_duration_seconds_sum",
		"tikv_coprocessor_request_duration_seconds_bucket",
		"tikv_coprocessor_request_duration_seconds_count",
		"tikv_coprocessor_request_duration_seconds_sum",
		"tikv_coprocessor_request_error",
		"tikv_coprocessor_request_handle_seconds_bucket",
		"tikv_coprocessor_request_handle_seconds_count",
		"tikv_coprocessor_request_handle_seconds_sum",
		"tikv_coprocessor_request_handler_build_seconds_bucket",
		"tikv_coprocessor_request_handler_build_seconds_count",
		"tikv_coprocessor_request_handler_build_seconds_sum",
		"tikv_coprocessor_request_wait_seconds_bucket",
		"tikv_coprocessor_request_wait_seconds_count",
		"tikv_coprocessor_request_wait_seconds_sum",
		"tikv_coprocessor_response_bytes",
		"tikv_coprocessor_rocksdb_perf",
		"tikv_coprocessor_scan_details",
		"tikv_coprocessor_scan_keys_bucket",
		"tikv_coprocessor_scan_keys_count",
		"tikv_coprocessor_scan_keys_sum",
		"tikv_engine_blob_cache_size_bytes",
		"tikv_engine_block_cache_size_bytes",
		"tikv_engine_bloom_efficiency",
		"tikv_engine_bytes_compressed",
		"tikv_engine_bytes_decompressed",
		"tikv_engine_bytes_per_read",
		"tikv_engine_bytes_per_write",
		"tikv_engine_cache_efficiency",
		"tikv_engine_compaction_flow_bytes",
		"tikv_engine_compaction_key_drop",
		"tikv_engine_compaction_outfile_sync_micro_seconds",
		"tikv_engine_compaction_time",
		"tikv_engine_compression_time_nanos",
		"tikv_engine_decompression_time_nanos",
		"tikv_engine_estimate_num_keys",
		"tikv_engine_file_status",
		"tikv_engine_flow_bytes",
		"tikv_engine_get_micro_seconds",
		"tikv_engine_get_served",
		"tikv_engine_hard_rate_limit_delay_count",
		"tikv_engine_locate",
		"tikv_engine_manifest_file_sync_micro_seconds",
		"tikv_engine_memory_bytes",
		"tikv_engine_memtable_efficiency",
		"tikv_engine_num_files_at_level",
		"tikv_engine_num_files_in_single_compaction",
		"tikv_engine_num_immutable_mem_table",
		"tikv_engine_num_snapshots",
		"tikv_engine_num_subcompaction_scheduled",
		"tikv_engine_oldest_snapshot_duration",
		"tikv_engine_pending_compaction_bytes",
		"tikv_engine_read_amp_flow_bytes",
		"tikv_engine_seek_micro_seconds",
		"tikv_engine_size_bytes",
		"tikv_engine_soft_rate_limit_delay_count",
		"tikv_engine_sst_read_micros",
		"tikv_engine_stall_l0_num_files_count",
		"tikv_engine_stall_l0_slowdown_count",
		"tikv_engine_stall_memtable_compaction_count",
		"tikv_engine_stall_micro_seconds",
		"tikv_engine_table_sync_micro_seconds",
		"tikv_engine_wal_file_sync_micro_seconds",
		"tikv_engine_wal_file_synced",
		"tikv_engine_write_micro_seconds",
		"tikv_engine_write_served",
		"tikv_engine_write_stall",
		"tikv_engine_write_stall_reason",
		"tikv_engine_write_wal_time_micro_seconds",
		"tikv_futurepool_handled_task_total",
		"tikv_futurepool_pending_task_total",
		"tikv_gcworker_autogc_processed_regions",
		"tikv_gcworker_autogc_safe_point",
		"tikv_gcworker_autogc_status",
		"tikv_grpc_msg_duration_seconds_bucket",
		"tikv_grpc_msg_duration_seconds_count",
		"tikv_grpc_msg_duration_seconds_sum",
		"tikv_grpc_request_source_counter_vec",
		"tikv_grpc_request_source_duration_vec",
		"tikv_in_memory_pessimistic_locking",
		"tikv_io_bytes",
		"tikv_load_base_split_duration_seconds_bucket",
		"tikv_load_base_split_duration_seconds_count",
		"tikv_load_base_split_duration_seconds_sum",
		"tikv_lock_manager_detector_leader_heartbeat",
		"tikv_log_backup_enabled",
		"tikv_log_backup_interal_actor_acting_duration_sec_bucket",
		"tikv_log_backup_interal_actor_acting_duration_sec_count",
		"tikv_log_backup_interal_actor_acting_duration_sec_sum",
		"tikv_multilevel_level0_chance",
		"tikv_multilevel_level_elapsed",
		"tikv_pd_heartbeat_message_total",
		"tikv_pd_pending_heartbeat_total",
		"tikv_pd_pending_tso_request_total",
		"tikv_pd_reconnect_total",
		"tikv_pd_request_duration_seconds_bucket",
		"tikv_pd_request_duration_seconds_count",
		"tikv_pd_request_duration_seconds_sum",
		"tikv_pending_delete_ranges_of_stale_peer",
		"tikv_pessimistic_lock_memory_size",
		"tikv_query_region_bucket",
		"tikv_query_region_count",
		"tikv_query_region_sum",
		"tikv_raft_entries_caches",
		"tikv_raftstore_admin_cmd_total",
		"tikv_raftstore_append_log_duration_seconds_bucket",
		"tikv_raftstore_append_log_duration_seconds_count",
		"tikv_raftstore_append_log_duration_seconds_sum",
		"tikv_raftstore_apply_duration_secs_bucket",
		"tikv_raftstore_apply_duration_secs_count",
		"tikv_raftstore_apply_duration_secs_sum",
		"tikv_raftstore_apply_log_duration_seconds_bucket",
		"tikv_raftstore_apply_log_duration_seconds_count",
		"tikv_raftstore_apply_log_duration_seconds_sum",
		"tikv_raftstore_apply_perf_context_time_duration_secs_bucket",
		"tikv_raftstore_apply_perf_context_time_duration_secs_count",
		"tikv_raftstore_apply_perf_context_time_duration_secs_sum",
		"tikv_raftstore_apply_proposal_bucket",
		"tikv_raftstore_apply_proposal_count",
		"tikv_raftstore_apply_proposal_sum",
		"tikv_raftstore_apply_wait_time_duration_secs_bucket",
		"tikv_raftstore_apply_wait_time_duration_secs_count",
		"tikv_raftstore_apply_wait_time_duration_secs_sum",
		"tikv_raftstore_check_split_total",
		"tikv_raftstore_commit_log_duration_seconds_bucket",
		"tikv_raftstore_commit_log_duration_seconds_count",
		"tikv_raftstore_commit_log_duration_seconds_sum",
		"tikv_raftstore_entry_fetches",
		"tikv_raftstore_event_duration_bucket",
		"tikv_raftstore_event_duration_count",
		"tikv_raftstore_event_duration_sum",
		"tikv_raftstore_gc_raft_log_total",
		"tikv_raftstore_hibernated_peer_state",
		"tikv_raftstore_inspect_duration_seconds_bucket",
		"tikv_raftstore_inspect_duration_seconds_count",
		"tikv_raftstore_inspect_duration_seconds_sum",
		"tikv_raftstore_leader_missing",
		"tikv_raftstore_local_read_cache_requests",
		"tikv_raftstore_local_read_executed_requests",
		"tikv_raftstore_local_read_executed_stale_read_requests",
		"tikv_raftstore_local_read_reject_total",
		"tikv_raftstore_local_read_renew_lease_advance_count",
		"tikv_raftstore_log_lag_bucket",
		"tikv_raftstore_log_lag_count",
		"tikv_raftstore_log_lag_sum",
		"tikv_raftstore_peer_msg_len_bucket",
		"tikv_raftstore_peer_msg_len_count",
		"tikv_raftstore_peer_msg_len_sum",
		"tikv_raftstore_proposal_total",
		"tikv_raftstore_propose_log_size_bucket",
		"tikv_raftstore_propose_log_size_count",
		"tikv_raftstore_propose_log_size_sum",
		"tikv_raftstore_raft_dropped_message_total",
		"tikv_raftstore_raft_invalid_proposal_total",
		"tikv_raftstore_raft_log_gc_deleted_keys_bucket",
		"tikv_raftstore_raft_log_gc_deleted_keys_count",
		"tikv_raftstore_raft_log_gc_deleted_keys_sum",
		"tikv_raftstore_raft_log_gc_seek_operations_count",
		"tikv_raftstore_raft_log_gc_skipped",
		"tikv_raftstore_raft_log_gc_write_duration_secs_bucket",
		"tikv_raftstore_raft_log_gc_write_duration_secs_count",
		"tikv_raftstore_raft_log_gc_write_duration_secs_sum",
		"tikv_raftstore_raft_log_kv_sync_duration_secs_bucket",
		"tikv_raftstore_raft_log_kv_sync_duration_secs_count",
		"tikv_raftstore_raft_log_kv_sync_duration_secs_sum",
		"tikv_raftstore_raft_process_duration_secs_bucket",
		"tikv_raftstore_raft_process_duration_secs_count",
		"tikv_raftstore_raft_process_duration_secs_sum",
		"tikv_raftstore_raft_ready_handled_total",
		"tikv_raftstore_raft_sent_message_total",
		"tikv_raftstore_read_index_pending",
		"tikv_raftstore_read_index_pending_duration_bucket",
		"tikv_raftstore_read_index_pending_duration_count",
		"tikv_raftstore_read_index_pending_duration_sum",
		"tikv_raftstore_region_count",
		"tikv_raftstore_region_keys_bucket",
		"tikv_raftstore_region_keys_count",
		"tikv_raftstore_region_keys_sum",
		"tikv_raftstore_region_size_bucket",
		"tikv_raftstore_region_size_count",
		"tikv_raftstore_region_size_sum",
		"tikv_raftstore_request_wait_time_duration_secs_bucket",
		"tikv_raftstore_request_wait_time_duration_secs_count",
		"tikv_raftstore_request_wait_time_duration_secs_sum",
		"tikv_raftstore_slow_score",
		"tikv_raftstore_snapshot_traffic_total",
		"tikv_raftstore_store_duration_secs_bucket",
		"tikv_raftstore_store_duration_secs_count",
		"tikv_raftstore_store_duration_secs_sum",
		"tikv_raftstore_store_wf_batch_wait_duration_seconds_bucket",
		"tikv_raftstore_store_wf_batch_wait_duration_seconds_count",
		"tikv_raftstore_store_wf_batch_wait_duration_seconds_sum",
		"tikv_raftstore_store_wf_before_write_duration_seconds_bucket",
		"tikv_raftstore_store_wf_before_write_duration_seconds_count",
		"tikv_raftstore_store_wf_before_write_duration_seconds_sum",
		"tikv_raftstore_store_wf_commit_log_duration_seconds_bucket",
		"tikv_raftstore_store_wf_commit_log_duration_seconds_count",
		"tikv_raftstore_store_wf_commit_log_duration_seconds_sum",
		"tikv_raftstore_store_wf_commit_not_persist_log_duration_seconds_bucket",
		"tikv_raftstore_store_wf_commit_not_persist_log_duration_seconds_count",
		"tikv_raftstore_store_wf_commit_not_persist_log_duration_seconds_sum",
		"tikv_raftstore_store_wf_persist_duration_seconds_bucket",
		"tikv_raftstore_store_wf_persist_duration_seconds_count",
		"tikv_raftstore_store_wf_persist_duration_seconds_sum",
		"tikv_raftstore_store_wf_send_proposal_duration_seconds_bucket",
		"tikv_raftstore_store_wf_send_proposal_duration_seconds_count",
		"tikv_raftstore_store_wf_send_proposal_duration_seconds_sum",
		"tikv_raftstore_store_wf_send_to_queue_duration_seconds_bucket",
		"tikv_raftstore_store_wf_send_to_queue_duration_seconds_count",
		"tikv_raftstore_store_wf_send_to_queue_duration_seconds_sum",
		"tikv_raftstore_store_wf_write_end_duration_seconds_bucket",
		"tikv_raftstore_store_wf_write_end_duration_seconds_count",
		"tikv_raftstore_store_wf_write_end_duration_seconds_sum",
		"tikv_raftstore_store_wf_write_kvdb_end_duration_seconds_bucket",
		"tikv_raftstore_store_wf_write_kvdb_end_duration_seconds_count",
		"tikv_raftstore_store_wf_write_kvdb_end_duration_seconds_sum",
		"tikv_raftstore_store_write_msg_block_wait_duration_seconds_bucket",
		"tikv_raftstore_store_write_msg_block_wait_duration_seconds_count",
		"tikv_raftstore_store_write_msg_block_wait_duration_seconds_sum",
		"tikv_raftstore_store_write_raftdb_duration_seconds_bucket",
		"tikv_raftstore_store_write_raftdb_duration_seconds_count",
		"tikv_raftstore_store_write_raftdb_duration_seconds_sum",
		"tikv_raftstore_store_write_send_duration_seconds_bucket",
		"tikv_raftstore_store_write_send_duration_seconds_count",
		"tikv_raftstore_store_write_send_duration_seconds_sum",
		"tikv_raftstore_store_write_task_wait_duration_secs_bucket",
		"tikv_raftstore_store_write_task_wait_duration_secs_count",
		"tikv_raftstore_store_write_task_wait_duration_secs_sum",
		"tikv_raftstore_write_cmd_total",
		"tikv_rate_limiter_max_bytes_per_sec",
		"tikv_read_qps_topn",
		"tikv_region_read_bytes_bucket",
		"tikv_region_read_bytes_count",
		"tikv_region_read_bytes_sum",
		"tikv_region_read_keys_bucket",
		"tikv_region_read_keys_count",
		"tikv_region_read_keys_sum",
		"tikv_region_written_bytes_bucket",
		"tikv_region_written_bytes_count",
		"tikv_region_written_bytes_sum",
		"tikv_region_written_keys_bucket",
		"tikv_region_written_keys_count",
		"tikv_region_written_keys_sum",
		"tikv_resolved_ts_channel_pending_cmd_bytes_total",
		"tikv_resolved_ts_check_leader_duration_seconds_bucket",
		"tikv_resolved_ts_check_leader_duration_seconds_count",
		"tikv_resolved_ts_check_leader_duration_seconds_sum",
		"tikv_resolved_ts_lock_heap_bytes",
		"tikv_resolved_ts_min_leader_resolved_ts",
		"tikv_resolved_ts_min_leader_resolved_ts_gap_millis",
		"tikv_resolved_ts_min_leader_resolved_ts_region",
		"tikv_resolved_ts_min_resolved_ts",
		"tikv_resolved_ts_min_resolved_ts_gap_millis",
		"tikv_resolved_ts_min_resolved_ts_region",
		"tikv_resolved_ts_pending_count",
		"tikv_resolved_ts_region_resolve_status",
		"tikv_resolved_ts_scan_duration_seconds_bucket",
		"tikv_resolved_ts_scan_duration_seconds_count",
		"tikv_resolved_ts_scan_duration_seconds_sum",
		"tikv_resolved_ts_scan_tasks",
		"tikv_resolved_ts_zero_resolved_ts",
		"tikv_resource_metering_stat_task_count",
		"tikv_scheduler_command_duration_seconds_bucket",
		"tikv_scheduler_command_duration_seconds_count",
		"tikv_scheduler_command_duration_seconds_sum",
		"tikv_scheduler_commands_pri_total",
		"tikv_scheduler_contex_total",
		"tikv_scheduler_kv_command_key_read_bucket",
		"tikv_scheduler_kv_command_key_read_count",
		"tikv_scheduler_kv_command_key_read_sum",
		"tikv_scheduler_kv_command_key_write_bucket",
		"tikv_scheduler_kv_command_key_write_count",
		"tikv_scheduler_kv_command_key_write_sum",
		"tikv_scheduler_kv_scan_details",
		"tikv_scheduler_latch_wait_duration_seconds_bucket",
		"tikv_scheduler_latch_wait_duration_seconds_count",
		"tikv_scheduler_latch_wait_duration_seconds_sum",
		"tikv_scheduler_processing_read_duration_seconds_bucket",
		"tikv_scheduler_processing_read_duration_seconds_count",
		"tikv_scheduler_processing_read_duration_seconds_sum",
		"tikv_scheduler_stage_total",
		"tikv_scheduler_throttle_cf",
		"tikv_scheduler_write_flow",
		"tikv_scheduler_writing_bytes",
		"tikv_server_cpu_cores_quota",
		"tikv_server_grpc_req_batch_size_bucket",
		"tikv_server_grpc_req_batch_size_count",
		"tikv_server_grpc_req_batch_size_sum",
		"tikv_server_grpc_resp_batch_size_bucket",
		"tikv_server_grpc_resp_batch_size_count",
		"tikv_server_grpc_resp_batch_size_sum",
		"tikv_server_info",
		"tikv_server_mem_trace_sum",
		"tikv_server_memory_usage",
		"tikv_storage_check_mem_lock_duration_seconds_bucket",
		"tikv_storage_check_mem_lock_duration_seconds_count",
		"tikv_storage_check_mem_lock_duration_seconds_sum",
		"tikv_storage_command_total",
		"tikv_storage_engine_async_request_duration_seconds_bucket",
		"tikv_storage_engine_async_request_duration_seconds_count",
		"tikv_storage_engine_async_request_duration_seconds_sum",
		"tikv_storage_engine_async_request_total",
		"tikv_storage_mvcc_check_txn_status",
		"tikv_storage_mvcc_duplicate_cmd_counter",
		"tikv_storage_mvcc_prewrite_assertion_perf",
		"tikv_storage_rocksdb_perf",
		"tikv_store_size_bytes",
		"tikv_stream_metadata_operation_latency_bucket",
		"tikv_stream_metadata_operation_latency_count",
		"tikv_stream_metadata_operation_latency_sum",
		"tikv_thread_cpu_seconds_total",
		"tikv_threads_io_bytes_total",
		"tikv_threads_state",
		"tikv_tiflash_proxy_multilevel_level0_chance",
		"tikv_tiflash_proxy_multilevel_level_elapsed",
		"tikv_unified_read_pool_running_tasks",
		"tikv_unified_read_pool_thread_count",
		"tikv_worker_handled_task_total",
		"tikv_worker_pending_task_total",
		"tikv_yatp_pool_schedule_wait_duration_bucket",
		"tikv_yatp_pool_schedule_wait_duration_count",
		"tikv_yatp_pool_schedule_wait_duration_sum",
		"tikv_yatp_task_exec_duration_bucket",
		"tikv_yatp_task_exec_duration_count",
		"tikv_yatp_task_exec_duration_sum",
		"tikv_yatp_task_execute_times_bucket",
		"tikv_yatp_task_execute_times_count",
		"tikv_yatp_task_execute_times_sum",
		"tikv_yatp_task_poll_duration_bucket",
		"tikv_yatp_task_poll_duration_count",
		"tikv_yatp_task_poll_duration_sum",
	}, nil
}

func getSeriesNum(c *http.Client, promAddr, query string) (int, error) {
	resp, err := c.Get(
		fmt.Sprintf("http://%s/api/v1/series?match[]=%s", promAddr, query),
	)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode/100 != 2 {
		return 0, fmt.Errorf("fail to get series. Status Code %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	r := struct {
		Series []interface{} `json:"data"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, err
	}
	return len(r.Series), nil
}

func collectMetric(
	l *logprinter.Logger,
	c *http.Client,
	promAddr string,
	beginTime, endTime time.Time,
	mtc string,
	label map[string]string,
	resultDir string,
	speedlimit int,
	compress bool,
) {
	query := generateQueryWitLabel(mtc, label)
	l.Debugf("Querying series of %s...", mtc)
	series, err := getSeriesNum(c, promAddr, query)
	if err != nil {
		l.Errorf("%s", err)
		return
	}
	if series <= 0 {
		l.Debugf("metric %s has %d series, ignore", mtc, series)
		return
	}

	// split time into smaller ranges to avoid querying too many data in one request
	if speedlimit == 0 {
		speedlimit = 10000
	}
	block := 3600 * speedlimit / series
	if block > maxQueryRange {
		block = maxQueryRange
	}
	if block < minQueryRange {
		block = minQueryRange
	}

	l.Debugf("Dumping metric %s-%s-%s...", mtc, beginTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	for queryEnd := endTime; queryEnd.After(beginTime); queryEnd = queryEnd.Add(time.Duration(-block) * time.Second) {
		querySec := block
		queryBegin := queryEnd.Add(time.Duration(-block) * time.Second)
		if queryBegin.Before(beginTime) {
			querySec = int(queryEnd.Sub(beginTime).Seconds())
			queryBegin = beginTime
		}
		if err := tiuputils.Retry(
			func() error {
				resp, err := c.PostForm(
					fmt.Sprintf("http://%s/api/v1/query", promAddr),
					url.Values{
						"query": {fmt.Sprintf("%s[%ds]", query, querySec)},
						"time":  {queryEnd.Format(time.RFC3339)},
					},
				)
				if err != nil {
					l.Errorf("failed query metric %s: %s, retry...", mtc, err)
					return err
				}
				// Prometheus API response format is JSON. Every successful API request returns a 2xx status code.
				if resp.StatusCode/100 != 2 {
					l.Errorf("failed query metric %s: Status Code %d, retry...", mtc, resp.StatusCode)
				}
				defer resp.Body.Close()

				dst, err := os.Create(
					filepath.Join(
						resultDir, subdirMonitor, subdirMetrics, strings.ReplaceAll(promAddr, ":", "-"),
						fmt.Sprintf("%s_%s_%s.json", mtc, queryBegin.Format(time.RFC3339), queryEnd.Format(time.RFC3339)),
					),
				)
				if err != nil {
					l.Errorf("collect metric %s: %s, retry...", mtc, err)
				}
				defer dst.Close()

				var enc io.WriteCloser
				var n int64
				if compress {
					// compress the metric
					enc, err = zstd.NewWriter(dst)
					if err != nil {
						l.Errorf("failed compressing metric %s: %s, retry...\n", mtc, err)
						return err
					}
					defer enc.Close()
				} else {
					enc = dst
				}
				n, err = io.Copy(enc, resp.Body)
				if err != nil {
					l.Errorf("failed writing metric %s to file: %s, retry...\n", mtc, err)
					return err
				}
				l.Debugf(" Dumped metric %s from %s to %s (%d bytes)", mtc, queryBegin.Format(time.RFC3339), queryEnd.Format(time.RFC3339), n)
				return nil
			},
			tiuputils.RetryOption{
				Attempts: 3,
				Delay:    time.Microsecond * 300,
				Timeout:  time.Second * 120,
			},
		); err != nil {
			l.Errorf("Error quering metrics %s: %s", mtc, err)
		}
	}
}

func ensureMonitorDir(base string, sub ...string) error {
	e := []string{base, subdirMonitor}
	e = append(e, sub...)
	dir := path.Join(e...)
	return os.MkdirAll(dir, 0755)
}

func filterMetrics(src, filter []string) []string {
	if filter == nil {
		return src
	}
	var res []string
	for _, metric := range src {
		for _, prefix := range filter {
			if strings.HasPrefix(metric, prefix) {
				res = append(res, metric)
			}
		}
	}
	return res
}

func generateQueryWitLabel(metric string, labels map[string]string) string {
	query := metric
	if len(labels) > 0 {
		query += "{"
		for k, v := range labels {
			query = fmt.Sprintf("%s%s=\"%s\",", query, k, v)
		}
		query = query[:len(query)-1] + "}"
	}
	return query
}

// TSDBCollectOptions is the options collecting TSDB file of prometheus, only work for tiup-cluster deployed cluster
type TSDBCollectOptions struct {
	*BaseOptions
	opt       *operator.Options // global operations from cli
	resultDir string
	fileStats map[string][]CollectStat
	compress  bool
	limit     int
}

// Desc implements the Collector interface
func (c *TSDBCollectOptions) Desc() string {
	return "metrics from Prometheus node"
}

// GetBaseOptions implements the Collector interface
func (c *TSDBCollectOptions) GetBaseOptions() *BaseOptions {
	return c.BaseOptions
}

// SetBaseOptions implements the Collector interface
func (c *TSDBCollectOptions) SetBaseOptions(opt *BaseOptions) {
	c.BaseOptions = opt
}

// SetGlobalOperations sets the global operation fileds
func (c *TSDBCollectOptions) SetGlobalOperations(opt *operator.Options) {
	c.opt = opt
}

// SetDir sets the result directory path
func (c *TSDBCollectOptions) SetDir(dir string) {
	c.resultDir = dir
}

// Prepare implements the Collector interface
func (c *TSDBCollectOptions) Prepare(m *Manager, cls *models.TiDBCluster) (map[string][]CollectStat, error) {
	if m.mode != CollectModeTiUP {
		return nil, nil
	}
	if len(cls.Monitors) < 1 {
		if m.logger.GetDisplayMode() == logprinter.DisplayModeDefault {
			fmt.Println("No Prometheus node found in topology, skip.")
		} else {
			m.logger.Warnf("No Prometheus node found in topology, skip.")
		}
		return nil, nil
	}

	// tsEnd, _ := utils.ParseTime(c.GetBaseOptions().ScrapeEnd)
	// tsStart, _ := utils.ParseTime(c.GetBaseOptions().ScrapeBegin)

	uniqueHosts := map[string]int{}             // host -> ssh-port
	uniqueArchList := make(map[string]struct{}) // map["os-arch"]{}
	hostPaths := make(map[string]set.StringSet)
	hostTasks := make(map[string]*task.Builder)

	topo := cls.Attributes[CollectModeTiUP].(spec.Topology)
	components := topo.ComponentsByUpdateOrder()
	var (
		dryRunTasks   []*task.StepDisplay
		downloadTasks []*task.StepDisplay
	)

	for _, comp := range components {
		if comp.Name() != spec.ComponentPrometheus {
			continue
		}

		for _, inst := range comp.Instances() {
			archKey := fmt.Sprintf("%s-%s", inst.OS(), inst.Arch())
			if _, found := uniqueArchList[archKey]; !found {
				uniqueArchList[archKey] = struct{}{}
				t0 := task.NewBuilder(m.logger).
					Download(
						componentDiagCollector,
						inst.OS(),
						inst.Arch(),
						"", // latest version
					).
					BuildAsStep(fmt.Sprintf("  - Downloading collecting tools for %s/%s", inst.OS(), inst.Arch()))
				downloadTasks = append(downloadTasks, t0)
			}

			// tasks that applies to each host
			if _, found := uniqueHosts[inst.GetHost()]; !found {
				uniqueHosts[inst.GetHost()] = inst.GetSSHPort()
				// build system info collecting tasks
				t1, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
				if err != nil {
					return nil, err
				}
				t1 = t1.
					Mkdir(c.GetBaseOptions().User, inst.GetHost(), filepath.Join(task.CheckToolsPathDir, "bin")).
					CopyComponent(
						componentDiagCollector,
						inst.OS(),
						inst.Arch(),
						"", // latest version
						"", // use default srcPath
						inst.GetHost(),
						task.CheckToolsPathDir,
					)
				hostTasks[inst.GetHost()] = t1
			}

			// add filepaths to list
			if _, found := hostPaths[inst.GetHost()]; !found {
				hostPaths[inst.GetHost()] = set.NewStringSet()
			}
			hostPaths[inst.GetHost()].Insert(inst.DataDir())
		}
	}

	// build scraper tasks
	for h, t := range hostTasks {
		host := h
		t = t.
			Shell(
				host,
				fmt.Sprintf("%s --prometheus '%s' -f '%s' -t '%s'",
					filepath.Join(task.CheckToolsPathDir, "bin", "scraper"),
					strings.Join(hostPaths[host].Slice(), ","),
					c.ScrapeBegin, c.ScrapeEnd,
				),
				"",
				false,
			).
			Func(
				host,
				func(ctx context.Context) error {
					stats, err := parseScraperSamples(ctx, host)
					if err != nil {
						return err
					}
					for host, files := range stats {
						c.fileStats[host] = files
					}
					return nil
				},
			)
		t1 := t.BuildAsStep(fmt.Sprintf("  - Scraping prometheus data files on %s:%d", host, uniqueHosts[host]))
		dryRunTasks = append(dryRunTasks, t1)
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Download necessary tools", false, downloadTasks...).
		ParallelStep("+ Collect host information", false, dryRunTasks...).
		Build()

	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)
	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return nil, err
		}
		return nil, perrs.Trace(err)
	}

	return c.fileStats, nil
}

// Collect implements the Collector interface
func (c *TSDBCollectOptions) Collect(m *Manager, cls *models.TiDBCluster) error {
	if m.mode != CollectModeTiUP {
		return nil
	}

	topo := cls.Attributes[CollectModeTiUP].(spec.Topology)
	var (
		collectTasks []*task.StepDisplay
		cleanTasks   []*task.StepDisplay
	)
	uniqueHosts := map[string]int{} // host -> ssh-port

	components := topo.ComponentsByUpdateOrder()

	for _, comp := range components {
		if comp.Name() != spec.ComponentPrometheus {
			continue
		}

		insts := comp.Instances()
		if len(insts) < 1 {
			return nil
		}

		// only collect from first promethes
		inst := insts[0]
		// checks that applies to each host
		if _, found := uniqueHosts[inst.GetHost()]; found {
			continue
		}
		uniqueHosts[inst.GetHost()] = inst.GetSSHPort()

		t2, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
		if err != nil {
			return err
		}
		for _, f := range c.fileStats[inst.GetHost()] {
			// build checking tasks
			t2 = t2.
				// check for listening ports
				CopyFile(
					f.Target,
					filepath.Join(c.resultDir, subdirMonitor, subdirRaw, fmt.Sprintf("%s-%d", inst.GetHost(), inst.GetMainPort()), filepath.Base(f.Target)),
					inst.GetHost(),
					true,
					c.limit,
					c.compress,
				)
		}
		collectTasks = append(
			collectTasks,
			t2.BuildAsStep(fmt.Sprintf("  - Downloading prometheus data files from node %s", inst.GetHost())),
		)

		b, err := m.sshTaskBuilder(c.GetBaseOptions().Cluster, topo, c.GetBaseOptions().User, *c.opt)
		if err != nil {
			return err
		}
		t3 := b.
			Rmdir(inst.GetHost(), task.CheckToolsPathDir).
			BuildAsStep(fmt.Sprintf("  - Cleanup temp files on %s:%d", inst.GetHost(), inst.GetSSHPort()))
		cleanTasks = append(cleanTasks, t3)
	}

	t := task.NewBuilder(m.logger).
		ParallelStep("+ Scrap files on nodes", false, collectTasks...).
		ParallelStep("+ Cleanup temp files", false, cleanTasks...).
		Build()

	ctx := ctxt.New(
		context.Background(),
		c.opt.Concurrency,
		m.logger,
	)
	if err := t.Execute(ctx); err != nil {
		if errorx.Cast(err) != nil {
			// FIXME: Map possible task errors and give suggestions.
			return err
		}
		return perrs.Trace(err)
	}

	return nil
}
