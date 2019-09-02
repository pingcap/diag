import _ from 'lodash';
import request from '@/utils/request';
import {
  bytesSizeFormatter,
  NumberConverer,
  toPercentUnit,
  toFixed,
  networkBitSizeFormatter,
  toFixed2,
  toPercent,
  toAnyUnit,
  toFixed1,
  timeSecondsFormatter,
} from '@/utils/formatter';

export interface IRawMetric {
  title?: string;
  promQLTemplate: string;
  labelTemplate: string;
  valConverter?: NumberConverer;
}

export interface IMetric {
  title?: string;
  promQL: string;
  labelTemplate: string;
  valConverter?: NumberConverer;
}

// https://www.lodashjs.com/docs/latest#_templatestring-options
// 使用自定义的模板分隔符
// _.templateSettings.interpolate = /{{([\s\S]+?)}}/g;
// var compiled = _.template('hello {{ user }}!');
// compiled({ 'user': 'mustache' });
// // => 'hello mustache!'

_.templateSettings.interpolate = /{{([\s\S]+?)}}/g;

// ////
const RAW_METRICS: { [key: string]: IRawMetric } = {
  // ////////////////////////////
  // Overview
  vcores_2: {
    promQLTemplate: 'count(node_cpu{mode="user", inspectionid="{{inspectionId}}"}) by (instance)',
    labelTemplate: '{{instance}}',
  },
  vcores_3: {
    promQLTemplate:
      'count(node_cpu_seconds_total{mode="user", inspectionid="{{inspectionId}}"}) by (instance)',
    labelTemplate: '{{instance}}',
  },

  memory_2: {
    promQLTemplate: 'node_memory_MemTotal{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{instance}}',
    valConverter: bytesSizeFormatter,
  },
  memory_3: {
    promQLTemplate: 'node_memory_MemTotal_bytes{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{ instance }}',
    valConverter: bytesSizeFormatter,
  },

  cpu_usage_2: {
    promQLTemplate:
      '100 - avg by (instance) (irate(node_cpu{mode="idle", inspectionid="{{inspectionId}}"}[1m]) ) * 100',
    labelTemplate: '{{instance}}',
    valConverter: val => toPercentUnit(val, 1),
  },
  cpu_usage_3: {
    promQLTemplate:
      '100 - avg by (instance) (irate(node_cpu_seconds_total{mode="idle", inspectionid="{{inspectionId}}"}[1m]) ) * 100',
    labelTemplate: '{{instance}}',
    valConverter: val => toPercentUnit(val, 1),
  },

  load: {
    promQLTemplate: 'node_load1{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{instance}}',
    valConverter: val => toFixed(val, 1),
  },

  memory_available_2: {
    promQLTemplate: 'node_memory_MemAvailable{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{instance}}',
    valConverter: bytesSizeFormatter,
  },
  memory_available_3: {
    promQLTemplate: 'node_memory_MemAvailable_bytes{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{instance}}',
    valConverter: bytesSizeFormatter,
  },

  network_traffic_receive_2: {
    promQLTemplate:
      'irate(node_network_receive_bytes{device!="lo", inspectionid="{{inspectionId}}"}[5m]) * 8',
    labelTemplate: 'Inbound: {{instance}}',
    valConverter: networkBitSizeFormatter,
  },
  network_traffic_transmit_2: {
    promQLTemplate:
      'irate(node_network_transmit_bytes{device!="lo", inspectionid="{{inspectionId}}"}[5m]) * 8',
    labelTemplate: 'Outbound: {{instance}}',
    valConverter: networkBitSizeFormatter,
  },
  network_traffic_receive_3: {
    promQLTemplate:
      'irate(node_network_receive_bytes_total{device!="lo", inspectionid="{{inspectionId}}"}[5m])',
    labelTemplate: 'Inbound: {{instance}}',
    valConverter: networkBitSizeFormatter,
  },
  network_traffic_transmit_3: {
    promQLTemplate:
      'irate(node_network_transmit_bytes_total{device!="lo", inspectionid="{{inspectionId}}"}[5m])',
    labelTemplate: 'Outbound: {{instance}}',
    valConverter: networkBitSizeFormatter,
  },

  tcp_retrans_syn: {
    promQLTemplate: 'irate(node_netstat_TcpExt_TCPSynRetrans{inspectionid="{{inspectionId}}"}[1m])',
    labelTemplate: '{{instance}} - TCPSynRetrans',
    valConverter: toFixed2,
  },
  tcp_retrans_slow_start: {
    promQLTemplate:
      'irate(node_netstat_TcpExt_TCPSlowStartRetrans{inspectionid="{{inspectionId}}"}[1m])',
    labelTemplate: '{{instance}} - TCPSlowStartRetrans',
    valConverter: toFixed2,
  },
  tcp_retrans_forward: {
    promQLTemplate:
      'irate(node_netstat_TcpExt_TCPForwardRetrans{inspectionid="{{inspectionId}}"}[1m])',
    labelTemplate: '{{instance}} - TCPForwardRetrans',
    valConverter: toFixed2,
  },
  tcp_retrans_3: {
    promQLTemplate: 'irate(node_netstat_Tcp_RetransSegs{inspectionid="{{inspectionId}}"}[1m])',
    labelTemplate: '{{instance}} - TCPSlowStartRetrans',
    valConverter: toFixed2,
  },

  io_util_2: {
    promQLTemplate: 'rate(node_disk_io_time_ms{inspectionid="{{inspectionId}}"}[1m]) / 1000',
    labelTemplate: '{{instance}} - {{device}}',
    valConverter: val => toPercent(val, 4),
  },
  io_util_3: {
    promQLTemplate: 'irate(node_disk_io_time_seconds_total{inspectionid="{{inspectionId}}"}[1m])',
    labelTemplate: '{{instance}} - {{device}}',
    valConverter: val => toPercent(val, 4),
  },

  // ////////////////////////////
  // PD
  // pd cluster
  disconnect_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_disconnected_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'Disconnect Stores',
  },
  unhealth_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_unhealth_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'Unhealth Stores',
  },
  low_space_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_low_space_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'LowSpace Stores',
  },
  down_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_down_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'Down Stores',
  },
  offline_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_offline_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'Offline Stores',
  },
  tombstone_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_tombstone_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'Tombstone Stores',
  },

  storage_capacity: {
    promQLTemplate:
      'sum(pd_cluster_status{type="storage_capacity", inspectionid="{{inspectionId}}" })',
    labelTemplate: 'storage capacity',
    valConverter: bytesSizeFormatter,
  },

  storage_size: {
    promQLTemplate: 'pd_cluster_status{type="storage_size", inspectionid="{{inspectionId}}"}',
    labelTemplate: 'storage size',
    valConverter: val => bytesSizeFormatter(val, true, 2),
  },
  storage_size_ratio: {
    promQLTemplate:
      'avg(pd_cluster_status{type="storage_size", inspectionid="{{inspectionId}}"}) / avg(pd_cluster_status{type="storage_capacity", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'used ratio',
    valConverter: val => toPercent(val, 6),
  },

  regions_label_level: {
    promQLTemplate: 'pd_regions_label_level{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{type}}',
  },

  regions_status: {
    promQLTemplate: 'pd_regions_status{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{type}}',
  },
  regions_status_sum: {
    promQLTemplate: 'sum(pd_regions_status{inspectionid="{{inspectionId}}"}) by (instance, type)',
    labelTemplate: '{{type}}',
  },

  // balance
  store_available: {
    promQLTemplate: '{inspectionid="{{inspectionId}}", type="store_available"}',
    labelTemplate: 'store-{{store}}',
    valConverter: val => bytesSizeFormatter(val, true, 2),
  },
  store_available_ratio: {
    promQLTemplate:
      'sum(pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="store_available"}) by (address, store) / sum(pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="store_capacity"}) by (address, store)',
    labelTemplate: '{{address}}-store-{{store}}',
    valConverter: val => toPercent(val, 3),
  },
  store_leader_score: {
    promQLTemplate:
      'pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="leader_score"}',
    labelTemplate: 'tikv-{{store}}',
  },
  store_region_score: {
    promQLTemplate:
      'pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="region_score"}',
    labelTemplate: 'tikv-{{store}}',
  },
  store_leader_count: {
    promQLTemplate:
      'pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="leader_count"}',
    labelTemplate: 'tikv-{{store}}',
  },

  // hot region
  hot_write_region_leader_distribution: {
    promQLTemplate:
      'pd_hotspot_status{inspectionid="{{inspectionId}}",type="hot_write_region_as_leader"}',
    labelTemplate: '{{store}}',
  },
  hot_write_region_peer_distribution: {
    promQLTemplate:
      'pd_hotspot_status{inspectionid="{{inspectionId}}",type="hot_write_region_as_peer"}',
    labelTemplate: '{{store}}',
  },
  hot_read_region_leader_distribution: {
    promQLTemplate:
      'pd_hotspot_status{inspectionid="{{inspectionId}}",type="hot_read_region_as_leader"}',
    labelTemplate: '{{store}}',
  },

  // operator
  schedule_operator_create: {
    promQLTemplate:
      'sum(delta(pd_schedule_operators_count{inspectionid="{{inspectionId}}", event="create"}[1m])) by (type)',
    labelTemplate: '{{type}}',
  },
  schedule_operator_timeout: {
    promQLTemplate:
      'sum(delta(pd_schedule_operators_count{inspectionid="{{inspectionId}}", event="timeout"}[1m])) by (type)',
    labelTemplate: '{{type}}',
  },

  // etcd
  handle_txn_count: {
    promQLTemplate:
      'sum(rate(pd_txn_handle_txns_duration_seconds_count{inspectionid="{{inspectionId}}"}[5m])) by (instance, result)',
    labelTemplate: '{{instance}} : {{result}}',
    valConverter: toFixed2,
  },
  wal_fsync_duration_seconds_99: {
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(etcd_disk_wal_fsync_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[5m])) by (instance, le))',
    labelTemplate: '{{instance}}',
    valConverter: val => timeSecondsFormatter(val, 2),
  },

  // tidb
  handle_request_duration_seconds_bucket: {
    promQLTemplate:
      'histogram_quantile(0.98, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[30s])) by (type, le))',
    labelTemplate: '{{type}} 98th percentile',
    valConverter: val => timeSecondsFormatter(val, 2),
  },
  handle_request_duration_seconds_avg: {
    promQLTemplate:
      'avg(rate(pd_client_request_handle_requests_duration_seconds_sum{inspectionid="{{inspectionId}}"}[30s])) by (type) /  avg(rate(pd_client_request_handle_requests_duration_seconds_count{inspectionid="{{inspectionId}}"}[30s])) by (type)',
    labelTemplate: '{{type}} average',
    valConverter: val => timeSecondsFormatter(val, 2),
  },

  // heartbeat
  region_heartbeat_latency_99: {
    promQLTemplate:
      'round(histogram_quantile(0.99, sum(rate(pd_scheduler_region_heartbeat_latency_seconds_bucket{inspectionid="{{inspectionId}}"}[5m])) by (store, le)), 1000)',
    labelTemplate: 'store{{store}}',
    valConverter: val => timeSecondsFormatter(val, 1),
  },

  grpc_completed_commands_duration_99: {
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(grpc_server_handling_seconds_bucket{inspectionid="{{inspectionId}}"}[5m])) by (grpc_method, le))',
    labelTemplate: '{{grpc_method}}',
    valConverter: val => timeSecondsFormatter(val, 1),
  },

  // ///////////////////////
  // TiDB
  // Query Summary: QPS, QPS By Instance, Duration, Failed Query OPM
  // qps
  qps_total: {
    promQLTemplate:
      'sum(rate(tidb_server_query_total{inspectionid="{{inspectionId}}"}[1m])) by (result)',
    labelTemplate: 'query {{result}}',
  },
  qps_total_yesterday: {
    promQLTemplate:
      'sum(rate(tidb_server_query_total{result="OK", inspectionid="{{inspectionId}}"}[1m]  offset 1d))',
    labelTemplate: 'yesterday',
  },
  qps_ideal: {
    promQLTemplate:
      'sum(tidb_server_connections{inspectionid="{{inspectionId}}"}) * sum(rate(tidb_server_handle_query_duration_seconds_count{inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tidb_server_handle_query_duration_seconds_sum{inspectionid="{{inspectionId}}"}[1m]))',
    labelTemplate: 'ideal QPS',
  },

  // qps by instance
  qps_by_instance: {
    promQLTemplate: 'rate(tidb_server_query_total{inspectionid="{{inspectionId}}"}[1m])',
    labelTemplate: '{{instance}} {{type}} {{result}}',
  },

  // duration
  duration_999: {
    promQLTemplate:
      'histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '999',
    valConverter: timeSecondsFormatter,
  },
  duration_99: {
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tidb_server_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99',
    valConverter: timeSecondsFormatter,
  },
  duration_95: {
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tidb_server_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '95',
    valConverter: timeSecondsFormatter,
  },
  duration_80: {
    promQLTemplate:
      'histogram_quantile(0.80, sum(rate(tidb_server_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '80',
    valConverter: timeSecondsFormatter,
  },

  // failed query opm
  failed_query_opm: {
    promQLTemplate:
      'sum(increase(tidb_server_execute_error_total{inspectionid="{{inspectionId}}"}[1m])) by (type, instance)',
    labelTemplate: '{{type}}-{{instance}}',
  },

  // slow query
  slow_query_process: {
    title: 'Slow query',
    promQLTemplate:
      'histogram_quantile(0.90, sum(rate(tidb_server_slow_query_process_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: 'all_proc',
    valConverter: timeSecondsFormatter,
  },
  slow_query_cop: {
    title: 'Slow query',
    promQLTemplate:
      'histogram_quantile(0.90, sum(rate(tidb_server_slow_query_cop_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: 'all_cop_proc',
  },
  slow_query_wait: {
    title: 'Slow query',
    promQLTemplate:
      'histogram_quantile(0.90, sum(rate(tidb_server_slow_query_wait_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: 'all_cop_wait',
  },

  // ////
  // Server Panel
  // uptime
  uptime: {
    title: 'Uptime',
    promQLTemplate: '(time() - process_start_time_seconds{job="tidb"})',
    labelTemplate: '{{instance}}',
    valConverter: val => timeSecondsFormatter(val, 1),
  },
  // cpu usage
  tidb_cpu_usage: {
    title: 'CPU Usage',
    promQLTemplate: 'rate(process_cpu_seconds_total{job="tidb"}[1m])',
    labelTemplate: '{{instance}}',
    valConverter: val => toPercent(val, 3),
  },
  // Connection Count
  connection_count: {
    promQLTemplate: 'tidb_server_connections{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{instance}}',
  },
  connection_count_sum: {
    promQLTemplate: 'sum(tidb_server_connections{inspectionid="{{inspectionId}}"})',
    labelTemplate: 'total',
  },
  // Goroutine Count
  goroutine_count: {
    promQLTemplate: ' go_goroutines{job=~"tidb.*", inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{instance}}',
  },
  heap_memory_usage: {
    promQLTemplate: 'go_memstats_heap_inuse_bytes{job=~"tidb.*", inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{instance}}',
    valConverter: bytesSizeFormatter,
  },
  // /////
  // Distsql Panel
  distsql_duration_999: {
    promQLTemplate:
      'histogram_quantile(0.999, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
    labelTemplate: '999-{{type}}',
    valConverter: val => timeSecondsFormatter(val, 1),
  },
  distsql_duration_99: {
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
    labelTemplate: '99-{{type}}',
  },
  distsql_duration_90: {
    promQLTemplate:
      'histogram_quantile(0.90, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
    labelTemplate: '90-{{type}}',
  },
  distsql_duration_50: {
    promQLTemplate:
      'histogram_quantile(0.50, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
    labelTemplate: '50-{{type}}',
  },
  // //////////
  // KV Errors Panel
  ticlient_region_error_total: {
    promQLTemplate:
      'sum(rate(tidb_tikvclient_region_err_total{inspectionid="{{inspectionId}}"}[1m])) by (type)',
    labelTemplate: '{{type}}',
  },
  ticlient_region_error_total_busy: {
    promQLTemplate:
      'sum(rate(tidb_tikvclient_region_err_total{type="server_is_busy", inspectionid="{{inspectionId}}"}[1m]))',
    labelTemplate: 'sum',
  },
  lock_resolve_ops: {
    promQLTemplate:
      'sum(rate(tidb_tikvclient_lock_resolver_actions_total{inspectionid="{{inspectionId}}"}[1m])) by (type)',
    labelTemplate: '{{type}}',
  },
  // ////////////
  // PD Client Panel
  pod_client_cmd_fail_ops: {
    promQLTemplate:
      'sum(rate(pd_client_cmd_handle_failed_cmds_duration_seconds_count{inspectionid="{{inspectionId}}"}[1m])) by (type)',
    labelTemplate: '{{type}}',
  },
  pd_tso_rpc_duration_999: {
    promQLTemplate:
      'histogram_quantile(0.999, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{type="tso", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '999',
    valConverter: val => timeSecondsFormatter(val, 1),
  },
  pd_tso_rpc_duration_99: {
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{type="tso", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99',
  },
  pd_tso_rpc_duration_90: {
    promQLTemplate:
      'histogram_quantile(0.90, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{type="tso", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '90',
  },

  // ///////////
  // Schema Load Panel
  load_schema_duration: {
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tidb_domain_load_schema_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, instance))',
    labelTemplate: '{{instance}}',
    valConverter: val => timeSecondsFormatter(val, 1),
  },
  schema_lease_error_opm: {
    promQLTemplate:
      'sum(increase(tidb_session_schema_lease_error_total{inspectionid="{{inspectionId}}"}[1m])) by (instance)',
    labelTemplate: '{{instance}}',
  },
  // /////////////
  // DDL Panel
  ddl_opm: {
    promQLTemplate:
      'increase(tidb_ddl_worker_operation_total{inspectionid="{{inspectionId}}"}[1m])',
    labelTemplate: '{{instance}}-{{type}}',
  },

  // ///////////////////
  // TiKV
  // Cluster: Store Size, CPU, Memory, IO Utilization, QPS, Leader
  tikv_store_size: {
    promQLTemplate: 'sum(tikv_engine_size_bytes{inspectionid="{{inspectionId}}"}) by (job)',
    labelTemplate: '{{job}}',
    valConverter: bytesSizeFormatter,
  },
  tikv_cpu: {
    promQLTemplate:
      'sum(rate(tikv_thread_cpu_seconds_total{inspectionid="{{inspectionId}}"}[1m])) by (instance,job)',
    labelTemplate: '{{instance}}-{{job}}',
    valConverter: val => toPercent(val, 3),
  },
  tikv_memory: {
    promQLTemplate: 'avg(process_resident_memory_bytes{inspectionid="{{inspectionId}}"}) by (job)',
    labelTemplate: '{{job}}',
    valConverter: bytesSizeFormatter,
  },
  tikv_io_utilization_2: {
    promQLTemplate: 'rate(node_disk_io_time_ms{inspectionid="{{inspectionId}}"}[1m]) / 1000',
    labelTemplate: '{{instance}} - {{device}}',
    valConverter: val => toPercent(val, 1),
  },
  tikv_io_utilization_3: {
    promQLTemplate: 'rate(node_disk_io_time_seconds_total{inspectionid="{{inspectionId}}"}[1m])',
    labelTemplate: '{{instance}} - {{device}}',
    valConverter: val => toPercent(val, 1),
  },

  tikv_qps: {
    title: 'QPS',
    promQLTemplate:
      'sum(rate(tikv_grpc_msg_duration_seconds_count{type!="kv_gc", inspectionid="{{inspectionId}}"}[1m])) by (job,type)',
    labelTemplate: '{{job}} - {{type}}',
    valConverter: val => toAnyUnit(val, 1, 1, 'ops'),
  },
  tikv_leader_2: {
    title: 'Leader',
    promQLTemplate:
      'sum(tikv_pd_heartbeat_tick_total{type="leader", inspectionid="{{inspectionId}}"}) by (job)',
    labelTemplate: '{{job}}',
  },
  tikv_leader_3: {
    title: 'Leader',
    promQLTemplate:
      'sum(tikv_raftstore_region_count{type="leader", inspectionid="{{inspectionId}}"}) by (instance)',
    labelTemplate: '{{instance}}',
  },

  // ///////////
  // Errors Panel
  tikv_server_busy_scheduler: {
    title: 'Server is Busy',
    promQLTemplate:
      'sum(rate(tikv_scheduler_too_busy_total{inspectionid="{{inspectionId}}"}[1m])) by (job)',
    labelTemplate: 'scheduler-{{job}}',
  },
  tikv_server_busy_channel: {
    title: 'Server is Busy',
    promQLTemplate:
      'sum(rate(tikv_channel_full_total{inspectionid="{{inspectionId}}"}[1m])) by (job, type)',
    labelTemplate: 'channelfull-{{job}}-{{type}}',
  },
  tikv_server_busy_coprocessor: {
    title: 'Server is Busy',
    promQLTemplate:
      'sum(rate(tikv_coprocessor_request_error{type="full", inspectionid="{{inspectionId}}"}[1m])) by (job)',
    labelTemplate: 'coprocessor-{{job}}',
  },
  tikv_server_busy_stall: {
    title: 'Server is Busy',
    promQLTemplate:
      'avg(tikv_engine_write_stall{type="write_stall_percentile99", inspectionid="{{inspectionId}}"}) by (job)',
    labelTemplate: 'stall-{{job}}',
  },

  tikv_server_report_failures: {
    title: 'Server Report Failures',
    promQLTemplate:
      'sum(rate(tikv_server_report_failure_msg_total{inspectionid="{{inspectionId}}"}[1m])) by (type,instance,job,store_id)',
    labelTemplate: '{{job}} - {{type}} - to - {{store_id}}',
  },
  tikv_raftstore_error: {
    title: 'Raftstore Error',
    promQLTemplate:
      'sum(rate(tikv_storage_engine_async_request_total{status!~"success|all", inspectionid="{{inspectionId}}"}[1m])) by (job, status)',
    labelTemplate: '{{job}}-{{status}}',
  },
  tikv_scheduler_error: {
    title: 'Scheduler Error',
    promQLTemplate:
      'sum(rate(tikv_scheduler_stage_total{stage=~"snapshot_err|prepare_write_err", inspectionid="{{inspectionId}}"}[1m])) by (job, stage)',
    labelTemplate: '{{job}}-{{stage}}',
  },
  tikv_coprocessor_error: {
    title: 'Coprocessor Error',
    promQLTemplate:
      'sum(rate(tikv_coprocessor_request_error{inspectionid="{{inspectionId}}"}[1m])) by (job, reason)',
    labelTemplate: '{{job}}-{{reason}}',
  },
  tikv_grpc_message_error: {
    title: 'gRPC message error',
    promQLTemplate:
      'sum(rate(tikv_grpc_msg_fail_total{inspectionid="{{inspectionId}}"}[1m])) by (job, type)',
    labelTemplate: '{{job}}-{{type}}',
  },
  tikv_leader_drop_2: {
    title: 'Leader drop',
    promQLTemplate:
      'sum(delta(tikv_pd_heartbeat_tick_total{type="leader", inspectionid="{{inspectionId}}"}[1m])) by (job)',
    labelTemplate: '{{job}}',
  },
  tikv_leader_drop_3: {
    title: 'Leader drop',
    promQLTemplate:
      'sum(delta(tikv_raftstore_region_count{type="leader", inspectionid="{{inspectionId}}"}[1m])) by (instance)',
    labelTemplate: '{{instance}}',
  },
  tikv_leader_missing: {
    title: 'Leader missing',
    promQLTemplate: 'sum(tikv_raftstore_leader_missing{inspectionid="{{inspectionId}}"}) by (job)',
    labelTemplate: '{{job}}',
  },

  // //////////////
  // Server Panel
  tikv_channel_full: {
    title: 'Channel full',
    promQLTemplate:
      'sum(rate(tikv_channel_full_total{inspectionid="{{inspectionId}}"}[1m])) by (job, type)',
    labelTemplate: '{{job}} - {{type}}',
  },

  tikv_approximate_region_size_99: {
    title: 'Approximate Region size',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_raftstore_region_size_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99%',
    valConverter: bytesSizeFormatter,
  },
  tikv_approximate_region_size_95: {
    title: 'Approximate Region size',
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_raftstore_region_size_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '95%',
  },
  tikv_approximate_region_size_avg: {
    title: 'Approximate Region size',
    promQLTemplate:
      'sum(rate(tikv_raftstore_region_size_sum{inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_raftstore_region_size_count{inspectionid="{{inspectionId}}"}[1m])) ',
    labelTemplate: 'avg',
  },

  // /////////////////
  // Raft IO Panel
  // Apply log duration
  tikv_apply_log_duration_99: {
    title: 'Apply log duration',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99%',
    valConverter: val => timeSecondsFormatter(val, 1),
  },
  tikv_apply_log_duration_90: {
    promQLTemplate:
      'histogram_quantile(0.90, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '90%',
  },
  tikv_apply_log_duration_avg: {
    promQLTemplate:
      'sum(rate(tikv_raftstore_apply_log_duration_seconds_sum{inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_raftstore_apply_log_duration_seconds_count{inspectionid="{{inspectionId}}"}[1m])) ',
    labelTemplate: 'avg',
  },

  tikv_apply_log_duration_per_server: {
    title: 'Apply log duration per server',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, job))',
    labelTemplate: '{{job}}',
    valConverter: val => timeSecondsFormatter(val, 1),
  },

  tikv_append_log_duration_99: {
    title: 'Append log duration',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99%',
    valConverter: val => timeSecondsFormatter(val, 1),
  },
  tikv_append_log_duration_95: {
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '95%',
  },
  tikv_append_log_duration_avg: {
    promQLTemplate:
      'sum(rate(tikv_raftstore_append_log_duration_seconds_sum{inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_raftstore_append_log_duration_seconds_count{inspectionid="{{inspectionId}}"}[1m])) ',
    labelTemplate: 'avg',
  },

  tikv_append_log_duration_per_server: {
    title: 'Append log duration per server',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, job))',
    labelTemplate: '{{job}}',
    valConverter: val => timeSecondsFormatter(val, 1),
  },

  // ////////////////
  // Scheduler - prewrite Panel
  tikv_scheduler_prewrite_latch_wait_duration_99: {
    title: 'Scheduler latch wait duration',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99%',
    valConverter: val => timeSecondsFormatter(val, 1),
  },
  tikv_scheduler_prewrite_latch_wait_duration_95: {
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '95%',
  },
  tikv_scheduler_prewrite_latch_wait_duration_avg: {
    promQLTemplate:
      'sum(rate(tikv_scheduler_latch_wait_duration_seconds_sum{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_scheduler_latch_wait_duration_seconds_count{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) ',
    labelTemplate: 'avg',
  },

  tivk_scheduler_prewrite_command_duration_99: {
    title: 'Scheduler command duration',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_scheduler_command_duration_seconds_bucket{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99%',
    valConverter: val => timeSecondsFormatter(val, 1),
  },
  tivk_scheduler_prewrite_command_duration_95: {
    title: 'Scheduler command duration',
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_scheduler_command_duration_seconds_bucket{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '95%',
  },
  tivk_scheduler_prewrite_command_duration_avg: {
    title: 'Scheduler command duration',
    promQLTemplate:
      'sum(rate(tikv_scheduler_command_duration_seconds_sum{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_scheduler_command_duration_seconds_count{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) ',
    labelTemplate: 'avg',
  },
  // ////////////////
  // Scheduler - commit Panel
  tikv_scheduler_commit_latch_wait_duration_99: {
    title: 'Scheduler latch wait duration',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{type="commit", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99%',
    valConverter: val => timeSecondsFormatter(val, 1),
  },
  tikv_scheduler_commit_latch_wait_duration_95: {
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{type="commit", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '95%',
  },
  tikv_scheduler_commit_latch_wait_duration_avg: {
    promQLTemplate:
      'sum(rate(tikv_scheduler_command_duration_seconds_sum{type="commit", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_scheduler_command_duration_seconds_count{type="commit", inspectionid="{{inspectionId}}"}[1m])) ',
    labelTemplate: 'avg',
  },

  tivk_scheduler_commit_command_duration_99: {
    title: 'Scheduler command duration',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_scheduler_command_duration_seconds_bucket{type="commit", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99%',
    valConverter: val => timeSecondsFormatter(val, 1),
  },
  tivk_scheduler_commit_command_duration_95: {
    title: 'Scheduler command duration',
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_scheduler_command_duration_seconds_bucket{type="commit", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '95%',
  },
  tivk_scheduler_commit_command_duration_avg: {
    title: 'Scheduler command duration',
    promQLTemplate:
      'sum(rate(tikv_scheduler_command_duration_seconds_sum{type="commit", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_scheduler_command_duration_seconds_count{type="commit", inspectionid="{{inspectionId}}"}[1m])) ',
    labelTemplate: 'avg',
  },
  // //////////////////////
  // Raft Propose Panel
  tikv_propose_wait_duration_99: {
    title: 'Propose wait duration',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_raftstore_request_wait_time_duration_secs_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99%',
    valConverter: val => timeSecondsFormatter(val, 0),
  },
  tikv_propose_wait_duration_95: {
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_raftstore_request_wait_time_duration_secs_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '95%',
  },
  tikv_propose_wait_duration_avg: {
    promQLTemplate:
      'sum(rate(tikv_raftstore_request_wait_time_duration_secs_sum{inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_raftstore_request_wait_time_duration_secs_count{inspectionid="{{inspectionId}}"}[1m]))',
    labelTemplate: 'avg',
  },
  // //////////////////////
  // Raft Message Panel
  tikv_raft_vote: {
    title: 'Vote',
    promQLTemplate:
      'sum(rate(tikv_raftstore_raft_sent_message_total{type="vote", inspectionid="{{inspectionId}}"}[1m])) by (job)',
    labelTemplate: '{{job}}',
    valConverter: val => toAnyUnit(val, 1, 1, 'ops'),
  },
  // //////////////////////
  // Storage Panel
  tikv_storage_async_write_duration_99: {
    title: 'Storage async write duration',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="write", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99%',
    valConverter: val => timeSecondsFormatter(val, 0),
  },
  tikv_storage_async_write_duration_95: {
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="write", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '95%',
  },
  tikv_storage_async_write_duration_avg: {
    promQLTemplate:
      'sum(rate(tikv_storage_engine_async_request_duration_seconds_sum{type="write", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_storage_engine_async_request_duration_seconds_count{type="write", inspectionid="{{inspectionId}}"}[1m]))',
    labelTemplate: 'avg',
  },

  tikv_storage_async_snapshot_duration_99: {
    title: 'Storage async snapshot duration',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="snapshot", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '99%',
    valConverter: val => timeSecondsFormatter(val, 0),
  },
  tikv_storage_async_snapshot_duration_95: {
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="snapshot", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: '95%',
  },
  tikv_storage_async_snapshot_duration_avg: {
    promQLTemplate:
      'sum(rate(tikv_storage_engine_async_request_duration_seconds_sum{type="snapshot", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_storage_engine_async_request_duration_seconds_count{type="write", inspectionid="{{inspectionId}}"}[1m]))',
    labelTemplate: 'avg',
  },

  // ///////////////////////
  // Scheduler Panel
  scheduler_pending_commands: {
    title: 'Scheduler pending commands',
    promQLTemplate: 'sum(tikv_scheduler_contex_total{inspectionid="{{inspectionId}}"}) by (job)',
    labelTemplate: '{{job}}',
  },

  // ///////////////////////
  // RocksDB - raft Panel
  // write duration
  rocksdb_raft_write_duration_max: {
    title: 'Write duration',
    promQLTemplate:
      'avg(tikv_engine_write_micro_seconds{db="raft",type="write_max", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'max',
    valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
  },
  rocksdb_raft_write_duration_99: {
    promQLTemplate:
      'avg(tikv_engine_write_micro_seconds{db="raft",type="write_percentile99", inspectionid="{{inspectionId}}"})',
    labelTemplate: '99%',
  },
  rocksdb_raft_write_duration_95: {
    promQLTemplate:
      'avg(tikv_engine_write_micro_seconds{db="raft",type="write_percentile95", inspectionid="{{inspectionId}}"})',
    labelTemplate: '95%',
  },
  rocksdb_raft_write_duration_avg: {
    promQLTemplate:
      'avg(tikv_engine_write_micro_seconds{db="raft",type="write_average", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'avg',
  },

  // write stall duration
  rocksdb_raft_write_stall_duration_max: {
    title: 'Write stall duration',
    promQLTemplate:
      'avg(tikv_engine_write_stall{db="raft",type="write_stall_max", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'max',
    valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
  },
  rocksdb_raft_write_stall_duration_99: {
    promQLTemplate:
      'avg(tikv_engine_write_stall{db="raft",type="write_stall_percentile99", inspectionid="{{inspectionId}}"})',
    labelTemplate: '99%',
  },
  rocksdb_raft_write_stall_duration_95: {
    promQLTemplate:
      'avg(tikv_engine_write_stall{db="raft",type="write_stall_percentile95", inspectionid="{{inspectionId}}"})',
    labelTemplate: '95%',
  },
  rocksdb_raft_write_stall_duration_avg: {
    promQLTemplate:
      'avg(tikv_engine_write_stall{db="raft",type="write_stall_average", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'avg',
  },

  // get duration
  rocksdb_raft_get_duration_max: {
    title: 'Get duration',
    promQLTemplate:
      'avg(tikv_engine_get_micro_seconds{db="raft",type="get_max", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'max',
    valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
  },
  rocksdb_raft_get_duration_99: {
    promQLTemplate:
      'avg(tikv_engine_get_micro_seconds{db="raft",type="get_percentile99", inspectionid="{{inspectionId}}"})',
    labelTemplate: '99%',
  },
  rocksdb_raft_get_duration_95: {
    promQLTemplate:
      'avg(tikv_engine_get_micro_seconds{db="raft",type="get_percentile95", inspectionid="{{inspectionId}}"})',
    labelTemplate: '95%',
  },
  rocksdb_raft_get_duration_avg: {
    promQLTemplate:
      'avg(tikv_engine_get_micro_seconds{db="raft",type="get_average", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'avg',
  },

  // seek duration
  rocksdb_raft_seek_duration_max: {
    title: 'Seek duration',
    promQLTemplate:
      'avg(tikv_engine_seek_micro_seconds{db="raft",type="seek_max", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'max',
    valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
  },
  rocksdb_raft_seek_duration_99: {
    promQLTemplate:
      'avg(tikv_engine_seek_micro_seconds{db="raft",type="seek_percentile99", inspectionid="{{inspectionId}}"})',
    labelTemplate: '99%',
  },
  rocksdb_raft_seek_duration_95: {
    promQLTemplate:
      'avg(tikv_engine_seek_micro_seconds{db="raft",type="seek_percentile95", inspectionid="{{inspectionId}}"})',
    labelTemplate: '95%',
  },
  rocksdb_raft_seek_duration_avg: {
    promQLTemplate:
      'avg(tikv_engine_seek_micro_seconds{db="raft",type="seek_average", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'avg',
  },

  // wal sync duration
  rocksdb_raft_wal_sync_duration_max: {
    title: 'WAL sync duration',
    promQLTemplate:
      'avg(tikv_engine_wal_file_sync_micro_seconds{db="raft",type="wal_file_sync_max", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'max',
    valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
  },
  rocksdb_raft_wal_sync_duration_99: {
    promQLTemplate:
      'avg(tikv_engine_wal_file_sync_micro_seconds{db="raft",type="wal_file_sync_percentile99", inspectionid="{{inspectionId}}"})',
    labelTemplate: '99%',
  },
  rocksdb_raft_wal_sync_duration_95: {
    promQLTemplate:
      'avg(tikv_engine_wal_file_sync_micro_seconds{db="raft",type="wal_file_sync_percentile95", inspectionid="{{inspectionId}}"})',
    labelTemplate: '95%',
  },
  rocksdb_raft_wal_sync_duration_avg: {
    promQLTemplate:
      'avg(tikv_engine_wal_file_sync_micro_seconds{db="raft",type="wal_file_sync_average", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'avg',
  },

  // wal sync operation
  rocksdb_raft_wal_sync_operations: {
    title: 'WAL sync operations',
    promQLTemplate:
      'sum(rate(tikv_engine_wal_file_synced{db="raft", inspectionid="{{inspectionId}}"}[1m]))',
    labelTemplate: 'sync',
    valConverter: val => toAnyUnit(val, 1, 1, 'ops'),
  },

  rocksdb_raft_number_files_each_level: {
    title: 'Number files at each level',
    promQLTemplate:
      'avg(tikv_engine_num_files_at_level{db="raft", inspectionid="{{inspectionId}}"}) by (cf, level)',
    labelTemplate: 'cf-{{cf}}, level-{{level}}',
    valConverter: toFixed1,
  },
  rocksdb_raft_compaction_pending_bytes: {
    title: 'Compaction pending bytes',
    promQLTemplate:
      'sum(rate(tikv_engine_pending_compaction_bytes{db="raft", inspectionid="{{inspectionId}}"}[1m])) by (cf)',
    labelTemplate: '{{cf}}',
  },
  rocksdb_raft_block_cache_size: {
    title: 'Block cache size',
    promQLTemplate:
      'avg(tikv_engine_block_cache_size_bytes{db="raft", inspectionid="{{inspectionId}}"}) by(cf)',
    labelTemplate: '{{cf}}',
    valConverter: bytesSizeFormatter,
  },

  // ///////////////////////
  // RocksDB - kv Panel
  // write duration
  rocksdb_kv_write_duration_max: {
    title: 'Write duration',
    promQLTemplate:
      'avg(tikv_engine_write_micro_seconds{db="kv",type="write_max", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'max',
    valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
  },
  rocksdb_kv_write_duration_99: {
    promQLTemplate:
      'avg(tikv_engine_write_micro_seconds{db="kv",type="write_percentile99", inspectionid="{{inspectionId}}"})',
    labelTemplate: '99%',
  },
  rocksdb_kv_write_duration_95: {
    promQLTemplate:
      'avg(tikv_engine_write_micro_seconds{db="kv",type="write_percentile95", inspectionid="{{inspectionId}}"})',
    labelTemplate: '95%',
  },
  rocksdb_kv_write_duration_avg: {
    promQLTemplate:
      'avg(tikv_engine_write_micro_seconds{db="kv",type="write_average", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'avg',
  },

  // write stall duration
  rocksdb_kv_write_stall_duration_max: {
    title: 'Write stall duration',
    promQLTemplate:
      'avg(tikv_engine_write_stall{db="kv",type="write_stall_max", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'max',
    valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
  },
  rocksdb_kv_write_stall_duration_99: {
    promQLTemplate:
      'avg(tikv_engine_write_stall{db="kv",type="write_stall_percentile99", inspectionid="{{inspectionId}}"})',
    labelTemplate: '99%',
  },
  rocksdb_kv_write_stall_duration_95: {
    promQLTemplate:
      'avg(tikv_engine_write_stall{db="kv",type="write_stall_percentile95", inspectionid="{{inspectionId}}"})',
    labelTemplate: '95%',
  },
  rocksdb_kv_write_stall_duration_avg: {
    promQLTemplate:
      'avg(tikv_engine_write_stall{db="kv",type="write_stall_average", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'avg',
  },

  // get duration
  rocksdb_kv_get_duration_max: {
    title: 'Get duration',
    promQLTemplate:
      'avg(tikv_engine_get_micro_seconds{db="kv",type="get_max", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'max',
    valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
  },
  rocksdb_kv_get_duration_99: {
    promQLTemplate:
      'avg(tikv_engine_get_micro_seconds{db="kv",type="get_percentile99", inspectionid="{{inspectionId}}"})',
    labelTemplate: '99%',
  },
  rocksdb_kv_get_duration_95: {
    promQLTemplate:
      'avg(tikv_engine_get_micro_seconds{db="kv",type="get_percentile95", inspectionid="{{inspectionId}}"})',
    labelTemplate: '95%',
  },
  rocksdb_kv_get_duration_avg: {
    promQLTemplate:
      'avg(tikv_engine_get_micro_seconds{db="kv",type="get_average", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'avg',
  },

  // seek duration
  rocksdb_kv_seek_duration_max: {
    title: 'Seek duration',
    promQLTemplate:
      'avg(tikv_engine_seek_micro_seconds{db="kv",type="seek_max", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'max',
    valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
  },
  rocksdb_kv_seek_duration_99: {
    promQLTemplate:
      'avg(tikv_engine_seek_micro_seconds{db="kv",type="seek_percentile99", inspectionid="{{inspectionId}}"})',
    labelTemplate: '99%',
  },
  rocksdb_kv_seek_duration_95: {
    promQLTemplate:
      'avg(tikv_engine_seek_micro_seconds{db="kv",type="seek_percentile95", inspectionid="{{inspectionId}}"})',
    labelTemplate: '95%',
  },
  rocksdb_kv_seek_duration_avg: {
    promQLTemplate:
      'avg(tikv_engine_seek_micro_seconds{db="kv",type="seek_average", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'avg',
  },

  // wal sync duration
  rocksdb_kv_wal_sync_duration_max: {
    title: 'WAL sync duration',
    promQLTemplate:
      'avg(tikv_engine_wal_file_sync_micro_seconds{db="kv",type="wal_file_sync_max", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'max',
    valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
  },
  rocksdb_kv_wal_sync_duration_99: {
    promQLTemplate:
      'avg(tikv_engine_wal_file_sync_micro_seconds{db="kv",type="wal_file_sync_percentile99", inspectionid="{{inspectionId}}"})',
    labelTemplate: '99%',
  },
  rocksdb_kv_wal_sync_duration_95: {
    promQLTemplate:
      'avg(tikv_engine_wal_file_sync_micro_seconds{db="kv",type="wal_file_sync_percentile95", inspectionid="{{inspectionId}}"})',
    labelTemplate: '95%',
  },
  rocksdb_kv_wal_sync_duration_avg: {
    promQLTemplate:
      'avg(tikv_engine_wal_file_sync_micro_seconds{db="kv",type="wal_file_sync_average", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'avg',
  },

  // wal sync operation
  rocksdb_kv_wal_sync_operations: {
    title: 'WAL sync operations',
    promQLTemplate:
      'sum(rate(tikv_engine_wal_file_synced{db="kv", inspectionid="{{inspectionId}}"}[1m]))',
    labelTemplate: 'sync',
    valConverter: val => toAnyUnit(val, 1, 1, 'ops'),
  },

  rocksdb_kv_number_files_each_level: {
    title: 'Number files at each level',
    promQLTemplate:
      'avg(tikv_engine_num_files_at_level{db="kv", inspectionid="{{inspectionId}}"}) by (cf, level)',
    labelTemplate: 'cf-{{cf}}, level-{{level}}',
    valConverter: toFixed1,
  },
  rocksdb_kv_compaction_pending_bytes: {
    title: 'Compaction pending bytes',
    promQLTemplate:
      'sum(rate(tikv_engine_pending_compaction_bytes{db="kv", inspectionid="{{inspectionId}}"}[1m])) by (cf)',
    labelTemplate: '{{cf}}',
  },
  rocksdb_kv_block_cache_size: {
    title: 'Block cache size',
    promQLTemplate:
      'avg(tikv_engine_block_cache_size_bytes{db="kv", inspectionid="{{inspectionId}}"}) by(cf)',
    labelTemplate: '{{cf}}',
    valConverter: bytesSizeFormatter,
  },
  // ////////////////////
  // Coprocessor Panel
  // request duration
  coprocessor_request_duration_99_99: {
    title: 'Request duration',
    promQLTemplate:
      'histogram_quantile(0.9999, sum(rate(tikv_coprocessor_request_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
    labelTemplate: '{{req}}-99.99%',
    valConverter: val => timeSecondsFormatter(val, 0),
  },
  coprocessor_request_duration_99: {
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_coprocessor_request_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
    labelTemplate: '{{req}}-99%',
  },
  coprocessor_request_duration_95: {
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
    labelTemplate: '{{req}}-95%',
  },
  coprocessor_request_duration_avg: {
    promQLTemplate:
      'sum(rate(tikv_coprocessor_request_duration_seconds_sum{inspectionid="{{inspectionId}}"}[1m])) by (req) / sum(rate(tikv_coprocessor_request_duration_seconds_count{inspectionid="{{inspectionId}}"}[1m])) by (req)',
    labelTemplate: '{{req}}-avg',
  },

  // wait duration
  coprocessor_wait_duration_99_99: {
    title: 'Wait duration',
    promQLTemplate:
      'histogram_quantile(0.9999, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
    labelTemplate: '{{req}}-99.99%',
    valConverter: val => timeSecondsFormatter(val, 0),
  },
  coprocessor_wait_duration_99: {
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
    labelTemplate: '{{req}}-99%',
  },
  coprocessor_wait_duration_95: {
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
    labelTemplate: '{{req}}-95%',
  },
  coprocessor_wait_duration_avg: {
    promQLTemplate:
      'sum(rate(tikv_coprocessor_request_wait_seconds_sum{inspectionid="{{inspectionId}}"}[1m])) by (req) / sum(rate(tikv_coprocessor_request_wait_seconds_count{inspectionid="{{inspectionId}}"}[1m])) by (req)',
    labelTemplate: '{{req}}-avg',
  },

  // scan keys
  coprocessor_scan_keys_99_99: {
    title: 'Scan keys',
    promQLTemplate:
      'histogram_quantile(0.9999, avg(rate(tikv_coprocessor_scan_keys_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, req))',
    labelTemplate: '{{req}}-99.99%',
  },
  coprocessor_scan_keys_99: {
    promQLTemplate:
      'histogram_quantile(0.99, avg(rate(tikv_coprocessor_scan_keys_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, req))',
    labelTemplate: '{{req}}-99%',
  },
  coprocessor_scan_keys_95: {
    promQLTemplate:
      'histogram_quantile(0.95, avg(rate(tikv_coprocessor_scan_keys_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, req))',
    labelTemplate: '{{req}}-95%',
  },
  coprocessor_scan_keys_90: {
    promQLTemplate:
      'histogram_quantile(0.90, avg(rate(tikv_coprocessor_scan_keys_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, req))',
    labelTemplate: '{{req}}-90%',
  },
  coprocessor_total_ops_details_table_scan: {
    title: 'Total Ops Details (Table Scan)',
    promQLTemplate:
      'sum(rate(tikv_coprocessor_scan_details{inspectionid="{{inspectionId}}", req="select"}[1m])) by (tag)',
    labelTemplate: '{{tag}}',
  },
  coprocessor_total_ops_details_index_scan: {
    title: 'Total Ops Details (Index Scan)',
    promQLTemplate:
      'sum(rate(tikv_coprocessor_scan_details{inspectionid="{{inspectionId}}", req="index"}[1m])) by (tag)',
    labelTemplate: '{{tag}}',
  },

  // 95% Coprocessor wait duration by store
  coprocessor_wait_duration_by_store_95: {
    title: '95% Wait duration by store',
    promQLTemplate:
      'histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, job,req))',
    labelTemplate: '{{job}}-{{req}}',
    valConverter: val => timeSecondsFormatter(val, 1),
  },

  // handle_snapshot_duration_99
  handle_snapshot_duration_99_send: {
    title: '99% Handle snapshot duration',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_server_send_snapshot_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: 'send',
    valConverter: val => timeSecondsFormatter(val, 1),
  },
  handle_snapshot_duration_99_apply: {
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_raftstore_snapshot_duration_seconds_bucket{type="apply", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: 'apply',
  },
  handle_snapshot_duration_99_generate: {
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_raftstore_snapshot_duration_seconds_bucket{type="generate", inspectionid="{{inspectionId}}"}[1m])) by (le))',
    labelTemplate: 'generate',
  },

  // thread cpu panel
  raft_store_cpu: {
    title: 'Raft store CPU',
    promQLTemplate:
      'sum(rate(tikv_thread_cpu_seconds_total{name=~"raftstore_.*", inspectionid="{{inspectionId}}"}[1m])) by (job, name)',
    labelTemplate: '{{job}}',
    valConverter: val => toPercent(val, 4),
  },
  async_apply_cpu_2: {
    title: 'Async apply CPU',
    promQLTemplate:
      'sum(rate(tikv_thread_cpu_seconds_total{name=~"apply_worker", inspectionid="{{inspectionId}}"}[1m])) by (job, name)',
    labelTemplate: '{{job}}',
    valConverter: val => toPercent(val, 4),
  },
  async_apply_cpu_3: {
    title: 'Async apply CPU',
    promQLTemplate:
      'sum(rate(tikv_thread_cpu_seconds_total{name=~"apply_[0-9]+", inspectionid="{{inspectionId}}"}[1m])) by (instance)',
    labelTemplate: '{{instance}}',
    valConverter: val => toPercent(val, 4),
  },
  coprocessor_cpu: {
    title: 'Coprocessor CPU',
    promQLTemplate:
      'sum(rate(tikv_thread_cpu_seconds_total{name=~"cop_.*", inspectionid="{{inspectionId}}"}[1m])) by (job)',
    labelTemplate: '{{job}}',
    valConverter: val => toPercent(val, 4),
  },
  storage_readpool_cpu: {
    title: 'Storage ReadPool CPU',
    promQLTemplate:
      'sum(rate(tikv_thread_cpu_seconds_total{name=~"store_read.*", inspectionid="{{inspectionId}}"}[1m])) by (job)',
    labelTemplate: '{{job}}',
    valConverter: val => toPercent(val, 4),
  },
  split_check_cpu: {
    title: 'Split check CPU',
    promQLTemplate:
      'sum(rate(tikv_thread_cpu_seconds_total{name=~"split_check", inspectionid="{{inspectionId}}"}[1m])) by (job)',
    labelTemplate: '{{job}}',
    valConverter: val => toPercent(val, 4),
  },
  grpc_poll_cpu: {
    title: 'gPRC poll CPU',
    promQLTemplate:
      'sum(rate(tikv_thread_cpu_seconds_total{name=~"grpc.*", inspectionid="{{inspectionId}}"}[1m])) by (job)',
    labelTemplate: '{{job}}',
    valConverter: val => toPercent(val, 4),
  },
  scheduler_cpu: {
    title: 'Scheduler CPU',
    promQLTemplate:
      'sum(rate(tikv_thread_cpu_seconds_total{name=~"storage_schedul.*", inspectionid="{{inspectionId}}"}[1m])) by (job)',
    labelTemplate: '{{job}}',
    valConverter: val => toPercent(val, 4),
  },

  // gRPC
  grpc_message_duration_99: {
    title: '99% gRPC messge duration',
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(tikv_grpc_msg_duration_seconds_bucket{type!="kv_gc", inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
    labelTemplate: '{{type}}',
    valConverter: val => timeSecondsFormatter(val, 0),
  },
};

export const RAW_METRICS_ARR: { [key: string]: IRawMetric[] } = {
  ...Object.keys(RAW_METRICS).reduce((accu, curVal) => {
    accu[curVal] = [RAW_METRICS[curVal]];
    return accu;
  }, {}),

  network_traffic: [
    RAW_METRICS.network_traffic_receive_2,
    RAW_METRICS.network_traffic_transmit_2,
    RAW_METRICS.network_traffic_receive_3,
    RAW_METRICS.network_traffic_transmit_3,
  ],
  tcp_retrans: [
    RAW_METRICS.tcp_retrans_syn,
    RAW_METRICS.tcp_retrans_slow_start,
    RAW_METRICS.tcp_retrans_forward,
    RAW_METRICS.tcp_retrans_3,
  ],
  stores_status: [
    RAW_METRICS.disconnect_stores,
    RAW_METRICS.unhealth_stores,
    RAW_METRICS.low_space_stores,
    RAW_METRICS.down_stores,
    RAW_METRICS.offline_stores,
    RAW_METRICS.tombstone_stores,
  ],
  region_health: [RAW_METRICS.regions_status, RAW_METRICS.regions_status_sum],

  handle_request_duration_seconds: [
    RAW_METRICS.handle_request_duration_seconds_bucket,
    RAW_METRICS.handle_request_duration_seconds_avg,
  ],

  qps: [RAW_METRICS.qps_total, RAW_METRICS.qps_total_yesterday, RAW_METRICS.qps_ideal],
  duration: [
    RAW_METRICS.duration_999,
    RAW_METRICS.duration_99,
    RAW_METRICS.duration_95,
    RAW_METRICS.duration_80,
  ],
  connection_count_all: [RAW_METRICS.connection_count, RAW_METRICS.connection_count_sum],
  distsql_duration: [
    RAW_METRICS.distsql_duration_999,
    RAW_METRICS.distsql_duration_99,
    RAW_METRICS.distsql_duration_90,
    RAW_METRICS.distsql_duration_50,
  ],
  ticlient_region_error: [
    RAW_METRICS.ticlient_region_error_total,
    RAW_METRICS.ticlient_region_error_total_busy,
  ],
  pd_tso_rpc_duration: [
    RAW_METRICS.pd_tso_rpc_duration_999,
    RAW_METRICS.pd_tso_rpc_duration_99,
    RAW_METRICS.pd_tso_rpc_duration_90,
  ],
  tikv_server_busy: [
    RAW_METRICS.tikv_server_busy_scheduler,
    RAW_METRICS.tikv_server_busy_channel,
    RAW_METRICS.tikv_server_busy_coprocessor,
    RAW_METRICS.tikv_server_busy_stall,
  ],
  tikv_approximate_region_size: [
    RAW_METRICS.tikv_approximate_region_size_99,
    RAW_METRICS.tikv_approximate_region_size_95,
    RAW_METRICS.tikv_approximate_region_size_avg,
  ],
  tikv_apply_log_duration: [
    RAW_METRICS.tikv_apply_log_duration_99,
    RAW_METRICS.tikv_apply_log_duration_90,
    RAW_METRICS.tikv_apply_log_duration_avg,
  ],
  tikv_append_log_duration: [
    RAW_METRICS.tikv_append_log_duration_99,
    RAW_METRICS.tikv_append_log_duration_95,
    RAW_METRICS.tikv_append_log_duration_avg,
  ],
  tikv_scheduler_prewrite_latch_wait_duration: [
    RAW_METRICS.tikv_scheduler_prewrite_latch_wait_duration_99,
    RAW_METRICS.tikv_scheduler_prewrite_latch_wait_duration_95,
    RAW_METRICS.tikv_scheduler_prewrite_latch_wait_duration_avg,
  ],
  tivk_scheduler_prewrite_command_duration: [
    RAW_METRICS.tivk_scheduler_prewrite_command_duration_99,
    RAW_METRICS.tivk_scheduler_prewrite_command_duration_95,
    RAW_METRICS.tivk_scheduler_prewrite_command_duration_avg,
  ],
  tikv_scheduler_commit_latch_wait_duration: [
    RAW_METRICS.tikv_scheduler_commit_latch_wait_duration_99,
    RAW_METRICS.tikv_scheduler_commit_latch_wait_duration_95,
    RAW_METRICS.tikv_scheduler_commit_latch_wait_duration_avg,
  ],
  tivk_scheduler_commit_command_duration: [
    RAW_METRICS.tivk_scheduler_commit_command_duration_99,
    RAW_METRICS.tivk_scheduler_commit_command_duration_95,
    RAW_METRICS.tivk_scheduler_commit_command_duration_avg,
  ],
  tikv_propose_wait_duration: [
    RAW_METRICS.tikv_propose_wait_duration_99,
    RAW_METRICS.tikv_propose_wait_duration_95,
    RAW_METRICS.tikv_propose_wait_duration_avg,
  ],
  tikv_storage_async_write_duration: [
    RAW_METRICS.tikv_storage_async_write_duration_99,
    RAW_METRICS.tikv_storage_async_write_duration_95,
    RAW_METRICS.tikv_storage_async_write_duration_avg,
  ],
  tikv_storage_async_snapshot_duration: [
    RAW_METRICS.tikv_storage_async_snapshot_duration_99,
    RAW_METRICS.tikv_storage_async_snapshot_duration_95,
    RAW_METRICS.tikv_storage_async_snapshot_duration_avg,
  ],
  rocksdb_raft_write_duration: [
    RAW_METRICS.rocksdb_raft_write_duration_max,
    RAW_METRICS.rocksdb_raft_write_duration_99,
    RAW_METRICS.rocksdb_raft_write_duration_95,
    RAW_METRICS.rocksdb_raft_write_duration_avg,
  ],
  rocksdb_raft_write_stall_duration: [
    RAW_METRICS.rocksdb_raft_write_stall_duration_max,
    RAW_METRICS.rocksdb_raft_write_stall_duration_99,
    RAW_METRICS.rocksdb_raft_write_stall_duration_95,
    RAW_METRICS.rocksdb_raft_write_stall_duration_avg,
  ],
  rocksdb_raft_get_duration: [
    RAW_METRICS.rocksdb_raft_get_duration_max,
    RAW_METRICS.rocksdb_raft_get_duration_99,
    RAW_METRICS.rocksdb_raft_get_duration_95,
    RAW_METRICS.rocksdb_raft_get_duration_avg,
  ],
  rocksdb_raft_seek_duration: [
    RAW_METRICS.rocksdb_raft_seek_duration_max,
    RAW_METRICS.rocksdb_raft_seek_duration_99,
    RAW_METRICS.rocksdb_raft_seek_duration_95,
    RAW_METRICS.rocksdb_raft_seek_duration_avg,
  ],
  rocksdb_raft_wal_sync_duration: [
    RAW_METRICS.rocksdb_raft_wal_sync_duration_max,
    RAW_METRICS.rocksdb_raft_wal_sync_duration_99,
    RAW_METRICS.rocksdb_raft_wal_sync_duration_95,
    RAW_METRICS.rocksdb_raft_wal_sync_duration_avg,
  ],

  rocksdb_kv_write_duration: [
    RAW_METRICS.rocksdb_kv_write_duration_max,
    RAW_METRICS.rocksdb_kv_write_duration_99,
    RAW_METRICS.rocksdb_kv_write_duration_95,
    RAW_METRICS.rocksdb_kv_write_duration_avg,
  ],
  rocksdb_kv_write_stall_duration: [
    RAW_METRICS.rocksdb_kv_write_stall_duration_max,
    RAW_METRICS.rocksdb_kv_write_stall_duration_99,
    RAW_METRICS.rocksdb_kv_write_stall_duration_95,
    RAW_METRICS.rocksdb_kv_write_stall_duration_avg,
  ],
  rocksdb_kv_get_duration: [
    RAW_METRICS.rocksdb_kv_get_duration_max,
    RAW_METRICS.rocksdb_kv_get_duration_99,
    RAW_METRICS.rocksdb_kv_get_duration_95,
    RAW_METRICS.rocksdb_kv_get_duration_avg,
  ],
  rocksdb_kv_seek_duration: [
    RAW_METRICS.rocksdb_kv_seek_duration_max,
    RAW_METRICS.rocksdb_kv_seek_duration_99,
    RAW_METRICS.rocksdb_kv_seek_duration_95,
    RAW_METRICS.rocksdb_kv_seek_duration_avg,
  ],
  rocksdb_kv_wal_sync_duration: [
    RAW_METRICS.rocksdb_kv_wal_sync_duration_max,
    RAW_METRICS.rocksdb_kv_wal_sync_duration_99,
    RAW_METRICS.rocksdb_kv_wal_sync_duration_95,
    RAW_METRICS.rocksdb_kv_wal_sync_duration_avg,
  ],

  coprocessor_request_duration: [
    RAW_METRICS.coprocessor_request_duration_99_99,
    RAW_METRICS.coprocessor_request_duration_99,
    RAW_METRICS.coprocessor_request_duration_95,
    RAW_METRICS.coprocessor_request_duration_avg,
  ],
  coprocessor_wait_duration: [
    RAW_METRICS.coprocessor_wait_duration_99_99,
    RAW_METRICS.coprocessor_wait_duration_99,
    RAW_METRICS.coprocessor_wait_duration_95,
    RAW_METRICS.coprocessor_wait_duration_avg,
  ],
  coprocessor_scan_keys: [
    RAW_METRICS.coprocessor_scan_keys_99_99,
    RAW_METRICS.coprocessor_scan_keys_99,
    RAW_METRICS.coprocessor_scan_keys_95,
    RAW_METRICS.coprocessor_scan_keys_90,
  ],
  handle_snapshot_duration_99: [
    RAW_METRICS.handle_snapshot_duration_99_send,
    RAW_METRICS.handle_snapshot_duration_99_apply,
    RAW_METRICS.handle_snapshot_duration_99_generate,
  ],

  slow_query: [
    RAW_METRICS.slow_query_process,
    RAW_METRICS.slow_query_cop,
    RAW_METRICS.slow_query_wait,
  ],

  tikv_leader: [RAW_METRICS.tikv_leader_2, RAW_METRICS.tikv_leader_3],
  tikv_leader_drop: [RAW_METRICS.tikv_leader_drop_2, RAW_METRICS.tikv_leader_drop_3],
  vcores: [RAW_METRICS.vcores_2, RAW_METRICS.vcores_3],
  memory: [RAW_METRICS.memory_2, RAW_METRICS.memory_3],
  cpu_usage: [RAW_METRICS.cpu_usage_2, RAW_METRICS.cpu_usage_3],
  memory_available: [RAW_METRICS.memory_available_2, RAW_METRICS.memory_available_3],
  io_util: [RAW_METRICS.io_util_2, RAW_METRICS.io_util_3],
  tikv_io_utilization: [RAW_METRICS.tikv_io_utilization_2, RAW_METRICS.tikv_io_utilization_3],
  async_apply_cpu: [RAW_METRICS.async_apply_cpu_2, RAW_METRICS.async_apply_cpu_3],
};

export interface IPanel {
  title: string;
  expand: boolean;
  charts: string[];
}

export const PANELS: { [key: string]: IPanel } = {
  tikv_coprocessor: {
    title: 'Coprocessor',
    expand: false,
    charts: [
      'coprocessor_request_duration',
      'coprocessor_wait_duration',
      // 'coprocessor_scan_keys',
      'coprocessor_wait_duration_by_store_95',
      'coprocessor_total_ops_details_table_scan',
      'coprocessor_total_ops_details_index_scan',
    ],
  },
  tikv_snapshot: {
    title: 'Snapshot',
    expand: false,
    charts: ['handle_snapshot_duration_99'],
  },
  tikv_thread_cpu: {
    title: 'Thread CPU',
    expand: false,
    charts: [
      'raft_store_cpu',
      'async_apply_cpu',
      'coprocessor_cpu',
      'storage_readpool_cpu',
      'split_check_cpu',
      'grpc_poll_cpu',
      'scheduler_cpu',
    ],
  },
  tikv_grpc: {
    title: 'gRPC',
    expand: false,
    charts: ['grpc_message_duration_99'],
  },
};

// /////////////////////////////////////////////////////////////////////////////////////

export function fillPromQLTemplate(rawMetrics: IRawMetric[], inspectionId: string): IMetric[] {
  return rawMetrics.map(rawMetric => ({
    ...rawMetric,
    promQL: _.template(rawMetric.promQLTemplate)({ inspectionId }),
  }));
}

export interface IPromParams {
  start: number;
  end: number;
  step: number;
}

// request:
// http://localhost:3000/metric/api/v1/query_range?query=pd_cluster_status%7Btype%3D%22storage_size%22%7D&start=1560836237&end=1560836537&step=20
// response:
// {
//   "status": "success",
//   "data": {
//     "resultType": "matrix",
//     "result": [
//       {
//         "metric": {
//           "__name__": "pd_cluster_status",
//           "cluster": "2553e691-81de-438e-94e3-b67d39aaae52",
//           "instance": "tidb-default-pd-0",
//           "job": "tidb-cluster",
//           "kubernetes_namespace": "2553e691-81de-438e-94e3-b67d39aaae52",
//           "kubernetes_node": "172.16.4.96",
//           "kubernetes_pod_ip": "10.233.93.71",
//           "namespace": "global",
//           "type": "storage_size"
//         },
//         "values": [
//           [
//             1560836339,
//             "132132072"
//           ],
//           [
//             1560836359,
//             "132132072"
//           ],
//           ...
//         ]
//       }
//     ]
//   }
// }
// return :
// labels: ['timestamp', 'tidb-default-pd-0', 'tidb-default-pd-1']
// values: [
//   [1560836339, 132132072, 132132800],
//   [1560836359, 132132071, 132132801],
// ]
export async function prometheusRangeQuery(
  query: string,
  labelTemplate: string,
  promParmas: IPromParams,
): Promise<{ metricLabels: string[]; metricValues: number[][] }> {
  const params = {
    query,
    ...promParmas,
  };
  const res = await request('/metric/query_range', { params });

  const metricLabels: string[] = ['timestamp'];
  const metricValues: number[][] = [];

  if (res !== undefined) {
    const { data } = res;
    const metricValuesItemArrLength = data.result.length + 1;
    data.result.forEach((item: any, idx: number) => {
      let label = '';
      try {
        label = _.template(labelTemplate)(item.metric);
      } catch (err) {
        label = `instance-${idx + 1}`;
      }

      metricLabels.push(label);
      // item.values is a 2 demension array
      const curValuesArr = item.values;
      curValuesArr.forEach((arr: any[], arrIdx: number) => {
        if (metricValues[arrIdx] === undefined) {
          metricValues[arrIdx] = Array(metricValuesItemArrLength).fill(0);
          metricValues[arrIdx][0] = arr[0] * 1000; // convert seconds to milliseconds
        }
        let numVal = +arr[1];
        if (Number.isNaN(numVal)) {
          numVal = 0;
        }
        metricValues[arrIdx][idx + 1] = numVal; // convert string to number
      });
    });
  }
  return { metricLabels, metricValues };
}
