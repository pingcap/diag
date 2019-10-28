import _ from 'lodash';
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

// ///////////////////////////////////

// https://www.lodashjs.com/docs/latest#_templatestring-options
// 使用自定义的模板分隔符
// _.templateSettings.interpolate = /{{([\s\S]+?)}}/g;
// var compiled = _.template('hello {{ user }}!');
// compiled({ 'user': 'mustache' });
// // => 'hello mustache!'

_.templateSettings.interpolate = /{{([\s\S]+?)}}/g;

// ///////////////////////////////////

export interface IPromQueryTemplate {
  version?: 'all' | 'v_2_x' | 'v_3_x'; // default is 'all'
  promQLTemplate: string;
  labelTemplate: string;
  valConverter?: NumberConverer;
}

export interface IPromQuery {
  promQL: string;
  labelTemplate: string;
  valConverter?: NumberConverer;
}

export interface IPromChart {
  title: string;

  chartType?: 'line' | 'table'; // default is line
  tableColumns?: [string, string]; // for chartType is table

  queries: IPromQueryTemplate[];
}

export const PROM_CHARTS: { [key: string]: IPromChart } = {
  // ////////////////////////////
  // Overview
  vcores: {
    title: 'Vcores',
    chartType: 'table',
    tableColumns: ['Host', 'CPU Num'],
    queries: [
      {
        version: 'v_2_x',
        promQLTemplate:
          'count(node_cpu{mode="user", inspectionid="{{inspectionId}}"}) by (instance)',
        labelTemplate: '{{instance}}',
      },
      {
        version: 'v_3_x',
        promQLTemplate:
          'count(node_cpu_seconds_total{mode="user", inspectionid="{{inspectionId}}"}) by (instance)',
        labelTemplate: '{{instance}}',
      },
    ],
  },

  memory_total: {
    title: 'Memory Total',
    chartType: 'table',
    tableColumns: ['Host', 'Memory'],
    queries: [
      {
        version: 'v_2_x',
        promQLTemplate: 'node_memory_MemTotal{inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{instance}}',
        valConverter: bytesSizeFormatter,
      },
      {
        version: 'v_3_x',
        promQLTemplate: 'node_memory_MemTotal_bytes{inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{ instance }}',
        valConverter: bytesSizeFormatter,
      },
    ],
  },

  global_cpu_usage: {
    title: 'Global CPU Usage',
    queries: [
      {
        version: 'v_2_x',
        promQLTemplate:
          '100 - avg by (instance) (irate(node_cpu{mode="idle", inspectionid="{{inspectionId}}"}[1m]) ) * 100',
        labelTemplate: '{{instance}}',
        valConverter: val => toPercentUnit(val, 1),
      },
      {
        version: 'v_3_x',
        promQLTemplate:
          '100 - avg by (instance) (irate(node_cpu_seconds_total{mode="idle", inspectionid="{{inspectionId}}"}[1m]) ) * 100',
        labelTemplate: '{{instance}}',
        valConverter: val => toPercentUnit(val, 1),
      },
    ],
  },

  load_1: {
    title: 'Load[1m]',
    queries: [
      {
        promQLTemplate: 'node_load1{inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{instance}}',
        valConverter: val => toFixed(val, 1),
      },
    ],
  },
  load_5: {
    title: 'Load[5m]',
    queries: [
      {
        promQLTemplate: 'node_load5{inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{instance}}',
        valConverter: val => toFixed(val, 1),
      },
    ],
  },
  load_15: {
    title: 'Load[15m]',
    queries: [
      {
        promQLTemplate: 'node_load15{inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{instance}}',
        valConverter: val => toFixed(val, 1),
      },
    ],
  },

  memory_available: {
    title: 'Memory Available',
    queries: [
      {
        version: 'v_2_x',
        promQLTemplate: 'node_memory_MemAvailable{inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{instance}}',
        valConverter: bytesSizeFormatter,
      },
      {
        version: 'v_3_x',
        promQLTemplate: 'node_memory_MemAvailable_bytes{inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{instance}}',
        valConverter: bytesSizeFormatter,
      },
    ],
  },

  network_traffic: {
    title: 'Network Traffic',
    queries: [
      {
        version: 'v_2_x',
        promQLTemplate:
          'irate(node_network_receive_bytes{device!="lo", inspectionid="{{inspectionId}}"}[5m]) * 8',
        labelTemplate: 'Inbound: {{instance}}',
        valConverter: networkBitSizeFormatter,
      },
      {
        version: 'v_2_x',
        promQLTemplate:
          'irate(node_network_transmit_bytes{device!="lo", inspectionid="{{inspectionId}}"}[5m]) * 8',
        labelTemplate: 'Outbound: {{instance}}',
        valConverter: networkBitSizeFormatter,
      },
      {
        version: 'v_3_x',
        promQLTemplate:
          'irate(node_network_receive_bytes_total{device!="lo", inspectionid="{{inspectionId}}"}[5m])',
        labelTemplate: 'Inbound: {{instance}}',
        valConverter: networkBitSizeFormatter,
      },
      {
        version: 'v_3_x',
        promQLTemplate:
          'irate(node_network_transmit_bytes_total{device!="lo", inspectionid="{{inspectionId}}"}[5m])',
        labelTemplate: 'Outbound: {{instance}}',
        valConverter: networkBitSizeFormatter,
      },
    ],
  },

  tcp_retrans: {
    title: 'TCP Retrans',
    queries: [
      {
        version: 'v_2_x',
        promQLTemplate:
          'irate(node_netstat_TcpExt_TCPSynRetrans{inspectionid="{{inspectionId}}"}[1m])',
        labelTemplate: '{{instance}} - TCPSynRetrans',
        valConverter: toFixed2,
      },
      {
        version: 'v_2_x',
        promQLTemplate:
          'irate(node_netstat_TcpExt_TCPSlowStartRetrans{inspectionid="{{inspectionId}}"}[1m])',
        labelTemplate: '{{instance}} - TCPSlowStartRetrans',
        valConverter: toFixed2,
      },
      {
        version: 'v_2_x',
        promQLTemplate:
          'irate(node_netstat_TcpExt_TCPForwardRetrans{inspectionid="{{inspectionId}}"}[1m])',
        labelTemplate: '{{instance}} - TCPForwardRetrans',
        valConverter: toFixed2,
      },
      {
        version: 'v_3_x',
        promQLTemplate: 'irate(node_netstat_Tcp_RetransSegs{inspectionid="{{inspectionId}}"}[1m])',
        labelTemplate: '{{instance}} - TCPSlowStartRetrans',
        valConverter: toFixed2,
      },
    ],
  },

  io_util: {
    title: 'IO Util',
    queries: [
      {
        version: 'v_2_x',
        promQLTemplate: 'rate(node_disk_io_time_ms{inspectionid="{{inspectionId}}"}[1m]) / 1000',
        labelTemplate: '{{instance}} - {{device}}',
        valConverter: val => toPercent(val, 4),
      },
      {
        version: 'v_3_x',
        promQLTemplate:
          'irate(node_disk_io_time_seconds_total{inspectionid="{{inspectionId}}"}[1m])',
        labelTemplate: '{{instance}} - {{device}}',
        valConverter: val => toPercent(val, 4),
      },
    ],
  },

  // ////////////////////////////
  // PD
  // pd cluster
  stores_status: {
    title: 'Store Status',
    queries: [
      {
        promQLTemplate:
          'sum(pd_cluster_status{type="store_disconnected_count", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'Disconnect Stores',
      },

      {
        promQLTemplate:
          'sum(pd_cluster_status{type="store_unhealth_count", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'Unhealth Stores',
      },
      {
        promQLTemplate:
          'sum(pd_cluster_status{type="store_low_space_count", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'LowSpace Stores',
      },
      {
        promQLTemplate:
          'sum(pd_cluster_status{type="store_down_count", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'Down Stores',
      },
      {
        promQLTemplate:
          'sum(pd_cluster_status{type="store_offline_count", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'Offline Stores',
      },
      {
        promQLTemplate:
          'sum(pd_cluster_status{type="store_tombstone_count", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'Tombstone Stores',
      },
    ],
  },

  storage_capacity: {
    title: 'Store Capacity',
    queries: [
      {
        promQLTemplate:
          'sum(pd_cluster_status{type="storage_capacity", inspectionid="{{inspectionId}}" })',
        labelTemplate: 'storage capacity',
        valConverter: bytesSizeFormatter,
      },
    ],
  },

  storage_size: {
    title: 'Storage Size',
    queries: [
      {
        promQLTemplate: 'pd_cluster_status{type="storage_size", inspectionid="{{inspectionId}}"}',
        labelTemplate: 'storage size',
        valConverter: val => bytesSizeFormatter(val, true, 2),
      },
    ],
  },

  storage_size_ratio: {
    title: 'Storage Size Ratio',
    queries: [
      {
        promQLTemplate:
          'avg(pd_cluster_status{type="storage_size", inspectionid="{{inspectionId}}"}) / avg(pd_cluster_status{type="storage_capacity", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'used ratio',
        valConverter: val => toPercent(val, 2),
      },
    ],
  },

  regions_label_level: {
    title: 'Region Label Isolation Level',
    queries: [
      {
        promQLTemplate: 'pd_regions_label_level{inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{type}}',
      },
    ],
  },

  region_health: {
    title: 'Region Health',
    queries: [
      {
        promQLTemplate: 'pd_regions_status{inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{type}}',
      },
      {
        promQLTemplate:
          'sum(pd_regions_status{inspectionid="{{inspectionId}}"}) by (instance, type)',
        labelTemplate: '{{type}}',
      },
    ],
  },

  // balance
  store_available: {
    title: 'Store Available',
    queries: [
      {
        promQLTemplate: '{inspectionid="{{inspectionId}}", type="store_available"}',
        labelTemplate: 'store-{{store}}',
        valConverter: val => bytesSizeFormatter(val, true, 2),
      },
    ],
  },
  store_available_ratio: {
    title: 'Store available ratio',
    queries: [
      {
        promQLTemplate:
          'sum(pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="store_available"}) by (address, store) / sum(pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="store_capacity"}) by (address, store)',
        labelTemplate: '{{address}}-store-{{store}}',
        valConverter: val => toPercent(val, 3),
      },
    ],
  },
  store_leader_score: {
    title: 'Store Leader Score',
    queries: [
      {
        promQLTemplate:
          'pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="leader_score"}',
        labelTemplate: 'tikv-{{store}}',
      },
    ],
  },
  store_region_score: {
    title: 'Store Region Score',
    queries: [
      {
        promQLTemplate:
          'pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="region_score"}',
        labelTemplate: 'tikv-{{store}}',
      },
    ],
  },
  store_leader_count: {
    title: 'Store Leader Count',
    queries: [
      {
        promQLTemplate:
          'pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="leader_count"}',
        labelTemplate: 'tikv-{{store}}',
      },
    ],
  },

  // hot region
  hot_write_region_leader_distribution: {
    title: "Hot write region's leader distribution",
    queries: [
      {
        promQLTemplate:
          'pd_hotspot_status{inspectionid="{{inspectionId}}",type="hot_write_region_as_leader"}',
        labelTemplate: '{{store}}',
      },
    ],
  },
  hot_write_region_peer_distribution: {
    title: "Hot write region's peer distribution",
    queries: [
      {
        promQLTemplate:
          'pd_hotspot_status{inspectionid="{{inspectionId}}",type="hot_write_region_as_peer"}',
        labelTemplate: '{{store}}',
      },
    ],
  },
  hot_read_region_leader_distribution: {
    title: "Hot read region's leader distribution",
    queries: [
      {
        promQLTemplate:
          'pd_hotspot_status{inspectionid="{{inspectionId}}",type="hot_read_region_as_leader"}',
        labelTemplate: '{{store}}',
      },
    ],
  },

  // operator
  schedule_operator_create: {
    title: 'Schedule Operator Create',
    queries: [
      {
        promQLTemplate:
          'sum(delta(pd_schedule_operators_count{inspectionid="{{inspectionId}}", event="create"}[1m])) by (type)',
        labelTemplate: '{{type}}',
      },
    ],
  },
  schedule_operator_timeout: {
    title: 'Schedule Operator Timeout',
    queries: [
      {
        promQLTemplate:
          'sum(delta(pd_schedule_operators_count{inspectionid="{{inspectionId}}", event="timeout"}[1m])) by (type)',
        labelTemplate: '{{type}}',
      },
    ],
  },

  // etcd
  handle_txn_count: {
    title: 'handle transactions count',
    queries: [
      {
        promQLTemplate:
          'sum(rate(pd_txn_handle_txns_duration_seconds_count{inspectionid="{{inspectionId}}"}[5m])) by (instance, result)',
        labelTemplate: '{{instance}} : {{result}}',
        valConverter: toFixed2,
      },
    ],
  },
  wal_fsync_duration_seconds_99: {
    title: '99% WAL fsync duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(etcd_disk_wal_fsync_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[5m])) by (instance, le))',
        labelTemplate: '{{instance}}',
        valConverter: val => timeSecondsFormatter(val, 2),
      },
    ],
  },

  // tidb
  handle_request_duration_seconds: {
    title: 'PD TSO handle requests duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.98, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[30s])) by (type, le))',
        labelTemplate: '{{type}} 98th percentile',
        valConverter: val => timeSecondsFormatter(val, 2),
      },
      {
        promQLTemplate:
          'avg(rate(pd_client_request_handle_requests_duration_seconds_sum{inspectionid="{{inspectionId}}"}[30s])) by (type) /  avg(rate(pd_client_request_handle_requests_duration_seconds_count{inspectionid="{{inspectionId}}"}[30s])) by (type)',
        labelTemplate: '{{type}} average',
        valConverter: val => timeSecondsFormatter(val, 2),
      },
    ],
  },

  // heartbeat
  region_heartbeat_latency_99: {
    title: '99% region heartbeat latency',
    queries: [
      {
        promQLTemplate:
          'round(histogram_quantile(0.99, sum(rate(pd_scheduler_region_heartbeat_latency_seconds_bucket{inspectionid="{{inspectionId}}"}[5m])) by (store, le)), 1000)',
        labelTemplate: 'store{{store}}',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
    ],
  },

  // grpc
  grpc_completed_commands_duration_99: {
    title: '99% completed commands duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(grpc_server_handling_seconds_bucket{inspectionid="{{inspectionId}}"}[5m])) by (grpc_method, le))',
        labelTemplate: '{{grpc_method}}',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
    ],
  },

  // ///////////////////////
  // TiDB
  // Query Summary: QPS, QPS By Instance, Duration, Failed Query OPM
  // qps
  qps: {
    title: 'TiDB QPS',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tidb_server_query_total{inspectionid="{{inspectionId}}"}[1m])) by (result)',
        labelTemplate: 'query {{result}}',
      },
      {
        promQLTemplate:
          'sum(rate(tidb_server_query_total{result="OK", inspectionid="{{inspectionId}}"}[1m]  offset 1d))',
        labelTemplate: 'yesterday',
      },
      {
        promQLTemplate:
          'sum(tidb_server_connections{inspectionid="{{inspectionId}}"}) * sum(rate(tidb_server_handle_query_duration_seconds_count{inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tidb_server_handle_query_duration_seconds_sum{inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'ideal QPS',
      },
    ],
  },

  // qps by instance
  qps_by_instance: {
    title: 'QPS By Instance',
    queries: [
      {
        promQLTemplate: 'rate(tidb_server_query_total{inspectionid="{{inspectionId}}"}[1m])',
        labelTemplate: '{{instance}} {{type}} {{result}}',
      },
    ],
  },

  // duration
  tidb_duration: {
    title: 'TiDB Duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '999',
        valConverter: timeSecondsFormatter,
      },
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tidb_server_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99',
        valConverter: timeSecondsFormatter,
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tidb_server_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '95',
        valConverter: timeSecondsFormatter,
      },
      {
        promQLTemplate:
          'histogram_quantile(0.80, sum(rate(tidb_server_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '80',
        valConverter: timeSecondsFormatter,
      },
    ],
  },

  // failed query opm
  failed_query_opm: {
    title: 'Failed Query OPM',
    queries: [
      {
        promQLTemplate:
          'sum(increase(tidb_server_execute_error_total{inspectionid="{{inspectionId}}"}[1m])) by (type, instance)',
        labelTemplate: '{{type}}-{{instance}}',
      },
    ],
  },

  // slow query
  slow_query: {
    title: 'Slow query',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.90, sum(rate(tidb_server_slow_query_process_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: 'all_proc',
        valConverter: timeSecondsFormatter,
      },
      {
        promQLTemplate:
          'histogram_quantile(0.90, sum(rate(tidb_server_slow_query_cop_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: 'all_cop_proc',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.90, sum(rate(tidb_server_slow_query_wait_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: 'all_cop_wait',
      },
    ],
  },

  // ////
  // Server Panel
  // uptime
  uptime: {
    title: 'Uptime',
    queries: [
      {
        promQLTemplate:
          '(time() - process_start_time_seconds{job="tidb", inspectionid="{{inspectionId}}"})',
        labelTemplate: '{{instance}}',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
    ],
  },
  // cpu usage
  pd_cpu_usage: {
    title: 'PD CPU Usage',
    queries: [
      {
        promQLTemplate:
          'rate(process_cpu_seconds_total{job="pd", inspectionid="{{inspectionId}}"}[1m])',
        labelTemplate: '{{instance}}',
        valConverter: val => toPercent(val, 1),
      },
    ],
  },
  tidb_cpu_usage: {
    title: 'TiDB CPU Usage',
    queries: [
      {
        promQLTemplate:
          'rate(process_cpu_seconds_total{job="tidb", inspectionid="{{inspectionId}}"}[1m])',
        labelTemplate: '{{instance}}',
        valConverter: val => toPercent(val, 1),
      },
    ],
  },
  tikv_cpu_usage: {
    title: 'TiKV CPU Usage',
    queries: [
      {
        promQLTemplate:
          'rate(process_cpu_seconds_total{job="tikv", inspectionid="{{inspectionId}}"}[1m])',
        labelTemplate: '{{instance}}',
        valConverter: val => toPercent(val, 1),
      },
    ],
  },
  // Connection Count
  connection_count: {
    title: 'Connection Count',
    queries: [
      {
        promQLTemplate: 'tidb_server_connections{inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{instance}}',
      },
      {
        promQLTemplate: 'sum(tidb_server_connections{inspectionid="{{inspectionId}}"})',
        labelTemplate: 'total',
      },
    ],
  },
  // Goroutine Count
  goroutine_count: {
    title: 'Goroutine Count',
    queries: [
      {
        promQLTemplate: ' go_goroutines{job=~"tidb.*", inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{instance}}',
      },
    ],
  },

  // memory usage
  heap_memory_usage: {
    title: 'Memory Usage',
    queries: [
      {
        promQLTemplate:
          'go_memstats_heap_inuse_bytes{job=~"tidb.*", inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{instance}}',
        valConverter: bytesSizeFormatter,
      },
    ],
  },
  // /////
  // Distsql Panel
  distsql_duration: {
    title: 'Distsql Duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.999, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
        labelTemplate: '999-{{type}}',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
        labelTemplate: '99-{{type}}',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.90, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
        labelTemplate: '90-{{type}}',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.50, sum(rate(tidb_distsql_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
        labelTemplate: '50-{{type}}',
      },
    ],
  },

  // //////////
  // KV Errors Panel
  ticlient_region_error: {
    title: 'TiClient Region Error OPS',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tidb_tikvclient_region_err_total{inspectionid="{{inspectionId}}"}[1m])) by (type)',
        labelTemplate: '{{type}}',
      },
      {
        promQLTemplate:
          'sum(rate(tidb_tikvclient_region_err_total{type="server_is_busy", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'sum',
      },
    ],
  },

  lock_resolve_ops: {
    title: 'Lock Resolve OPS',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tidb_tikvclient_lock_resolver_actions_total{inspectionid="{{inspectionId}}"}[1m])) by (type)',
        labelTemplate: '{{type}}',
      },
    ],
  },
  // ////////////
  // PD Client Panel

  //
  pd_client_cmd_fail_ops: {
    title: 'PD Client CMD Fail OPS',
    queries: [
      {
        promQLTemplate:
          'sum(rate(pd_client_cmd_handle_failed_cmds_duration_seconds_count{inspectionid="{{inspectionId}}"}[1m])) by (type)',
        labelTemplate: '{{type}}',
      },
    ],
  },

  //
  pd_tso_rpc_duration: {
    title: 'PD TSO RPC Duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.999, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{type="tso", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '999',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{type="tso", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.90, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket{type="tso", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '90',
      },
    ],
  },

  // ///////////
  // Schema Load Panel
  load_schema_duration: {
    title: 'Load Schema Duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tidb_domain_load_schema_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, instance))',
        labelTemplate: '{{instance}}',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
    ],
  },
  schema_lease_error_opm: {
    title: 'Schema Lease Error OPM',
    queries: [
      {
        promQLTemplate:
          'sum(increase(tidb_session_schema_lease_error_total{inspectionid="{{inspectionId}}"}[1m])) by (instance)',
        labelTemplate: '{{instance}}',
      },
    ],
  },
  // /////////////
  // DDL Panel
  ddl_opm: {
    title: 'DDL META OPM',
    queries: [
      {
        promQLTemplate:
          'increase(tidb_ddl_worker_operation_total{inspectionid="{{inspectionId}}"}[1m])',
        labelTemplate: '{{instance}}-{{type}}',
      },
    ],
  },

  // ///////////////////
  // TiKV
  // Cluster: Store Size, CPU, Memory, IO Utilization, QPS, Leader
  tikv_store_size: {
    title: 'Store Size',
    queries: [
      {
        promQLTemplate: 'sum(tikv_engine_size_bytes{inspectionid="{{inspectionId}}"}) by (job)',
        labelTemplate: '{{job}}',
        valConverter: bytesSizeFormatter,
      },
    ],
  },

  tikv_thread_cpu: {
    title: 'TiKV Thread CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{inspectionid="{{inspectionId}}"}[1m])) by (instance,job)',
        labelTemplate: '{{instance}}-{{job}}',
        valConverter: val => toPercent(val, 3),
      },
    ],
  },

  tikv_memory: {
    title: 'Memory',
    queries: [
      {
        promQLTemplate:
          'avg(process_resident_memory_bytes{inspectionid="{{inspectionId}}"}) by (job)',
        labelTemplate: '{{job}}',
        valConverter: bytesSizeFormatter,
      },
    ],
  },

  tikv_io_utilization: {
    title: 'IO Utilization',
    queries: [
      {
        version: 'v_2_x',
        promQLTemplate: 'rate(node_disk_io_time_ms{inspectionid="{{inspectionId}}"}[1m]) / 1000',
        labelTemplate: '{{instance}} - {{device}}',
        valConverter: val => toPercent(val, 1),
      },
      {
        version: 'v_3_x',
        promQLTemplate:
          'rate(node_disk_io_time_seconds_total{inspectionid="{{inspectionId}}"}[1m])',
        labelTemplate: '{{instance}} - {{device}}',
        valConverter: val => toPercent(val, 1),
      },
    ],
  },

  tikv_qps: {
    title: 'TiKV QPS',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_grpc_msg_duration_seconds_count{type!="kv_gc", inspectionid="{{inspectionId}}"}[1m])) by (job,type)',
        labelTemplate: '{{job}} - {{type}}',
        valConverter: val => toAnyUnit(val, 1, 1, 'ops'),
      },
    ],
  },

  tikv_leader: {
    title: 'Leader',
    queries: [
      {
        version: 'v_2_x',
        promQLTemplate:
          'sum(tikv_pd_heartbeat_tick_total{type="leader", inspectionid="{{inspectionId}}"}) by (job)',
        labelTemplate: '{{job}}',
      },
      {
        version: 'v_3_x',
        promQLTemplate:
          'sum(tikv_raftstore_region_count{type="leader", inspectionid="{{inspectionId}}"}) by (instance)',
        labelTemplate: '{{instance}}',
      },
    ],
  },

  // ///////////
  // Errors Panel
  tikv_server_busy: {
    title: 'Server is Busy',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_scheduler_too_busy_total{inspectionid="{{inspectionId}}"}[1m])) by (job)',
        labelTemplate: 'scheduler-{{job}}',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_channel_full_total{inspectionid="{{inspectionId}}"}[1m])) by (job, type)',
        labelTemplate: 'channelfull-{{job}}-{{type}}',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_coprocessor_request_error{type="full", inspectionid="{{inspectionId}}"}[1m])) by (job)',
        labelTemplate: 'coprocessor-{{job}}',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_stall{type="write_stall_percentile99", inspectionid="{{inspectionId}}"}) by (job)',
        labelTemplate: 'stall-{{job}}',
      },
    ],
  },

  tikv_server_report_failures: {
    title: 'Server Report Failures',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_server_report_failure_msg_total{inspectionid="{{inspectionId}}"}[1m])) by (type,instance,job,store_id)',
        labelTemplate: '{{job}} - {{type}} - to - {{store_id}}',
      },
    ],
  },

  tikv_raftstore_error: {
    title: 'Raftstore Error',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_storage_engine_async_request_total{status!~"success|all", inspectionid="{{inspectionId}}"}[1m])) by (job, status)',
        labelTemplate: '{{job}}-{{status}}',
      },
    ],
  },
  tikv_scheduler_error: {
    title: 'Scheduler Error',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_scheduler_stage_total{stage=~"snapshot_err|prepare_write_err", inspectionid="{{inspectionId}}"}[1m])) by (job, stage)',
        labelTemplate: '{{job}}-{{stage}}',
      },
    ],
  },
  tikv_coprocessor_error: {
    title: 'Coprocessor Error',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_coprocessor_request_error{inspectionid="{{inspectionId}}"}[1m])) by (job, reason)',
        labelTemplate: '{{job}}-{{reason}}',
      },
    ],
  },
  tikv_grpc_message_error: {
    title: 'gRPC message error',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_grpc_msg_fail_total{inspectionid="{{inspectionId}}"}[1m])) by (job, type)',
        labelTemplate: '{{job}}-{{type}}',
      },
    ],
  },
  tikv_leader_drop: {
    title: 'Leader drop',
    queries: [
      {
        version: 'v_2_x',
        promQLTemplate:
          'sum(delta(tikv_pd_heartbeat_tick_total{type="leader", inspectionid="{{inspectionId}}"}[1m])) by (job)',
        labelTemplate: '{{job}}',
      },
      {
        version: 'v_3_x',
        promQLTemplate:
          'sum(delta(tikv_raftstore_region_count{type="leader", inspectionid="{{inspectionId}}"}[1m])) by (instance)',
        labelTemplate: '{{instance}}',
      },
    ],
  },
  tikv_leader_missing: {
    title: 'Leader missing',
    queries: [
      {
        promQLTemplate:
          'sum(tikv_raftstore_leader_missing{inspectionid="{{inspectionId}}"}) by (job)',
        labelTemplate: '{{job}}',
      },
    ],
  },

  // //////////////
  // Server Panel
  tikv_channel_full: {
    title: 'Channel full',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_channel_full_total{inspectionid="{{inspectionId}}"}[1m])) by (job, type)',
        labelTemplate: '{{job}} - {{type}}',
      },
    ],
  },

  tikv_approximate_region_size: {
    title: 'Approximate Region size',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_raftstore_region_size_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99%',
        valConverter: bytesSizeFormatter,
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_raftstore_region_size_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_raftstore_region_size_sum{inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_raftstore_region_size_count{inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'avg',
      },
    ],
  },

  // /////////////////
  // Raft IO Panel
  // Apply log duration
  tikv_apply_log_duration: {
    title: 'Apply log duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99%',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.90, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '90%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_raftstore_apply_log_duration_seconds_sum{inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_raftstore_apply_log_duration_seconds_count{inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'avg',
      },
    ],
  },

  tikv_apply_log_duration_per_server: {
    title: 'Apply log duration per server',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, job))',
        labelTemplate: '{{job}}',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
    ],
  },

  tikv_append_log_duration: {
    title: 'Append log duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99%',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_raftstore_append_log_duration_seconds_sum{inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_raftstore_append_log_duration_seconds_count{inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'avg',
      },
    ],
  },

  tikv_append_log_duration_per_server: {
    title: 'Append log duration per server',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, job))',
        labelTemplate: '{{job}}',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
    ],
  },

  // ////////////////
  // Scheduler - prewrite Panel
  tikv_scheduler_prewrite_latch_wait_duration: {
    title: 'Scheduler latch wait duration (prewrite)',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99%',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_scheduler_latch_wait_duration_seconds_sum{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_scheduler_latch_wait_duration_seconds_count{type="prewrite", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'avg',
      },
    ],
  },

  tivk_scheduler_prewrite_command_duration: {
    title: 'Scheduler command duration (prewrite)',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_scheduler_command_duration_seconds_bucket{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99%',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_scheduler_command_duration_seconds_bucket{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_scheduler_command_duration_seconds_sum{type="prewrite", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_scheduler_command_duration_seconds_count{type="prewrite", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'avg',
      },
    ],
  },
  // ////////////////
  // Scheduler - commit Panel
  tikv_scheduler_commit_latch_wait_duration: {
    title: 'Scheduler latch wait duration (commit)',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{type="commit", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99%',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{type="commit", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_scheduler_command_duration_seconds_sum{type="commit", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_scheduler_command_duration_seconds_count{type="commit", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'avg',
      },
    ],
  },

  tivk_scheduler_commit_command_duration: {
    title: 'Scheduler command duration (commit)',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_scheduler_command_duration_seconds_bucket{type="commit", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99%',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_scheduler_command_duration_seconds_bucket{type="commit", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_scheduler_command_duration_seconds_sum{type="commit", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_scheduler_command_duration_seconds_count{type="commit", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'avg',
      },
    ],
  },
  // //////////////////////
  // Raft Propose Panel
  tikv_propose_wait_duration: {
    title: 'Propose wait duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_raftstore_request_wait_time_duration_secs_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99%',
        valConverter: val => timeSecondsFormatter(val, 0),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_raftstore_request_wait_time_duration_secs_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_raftstore_request_wait_time_duration_secs_sum{inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_raftstore_request_wait_time_duration_secs_count{inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'avg',
      },
    ],
  },

  // //////////////////////
  // Raft Message Panel
  tikv_raft_vote: {
    title: 'Vote',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_raftstore_raft_sent_message_total{type="vote", inspectionid="{{inspectionId}}"}[1m])) by (job)',
        labelTemplate: '{{job}}',
        valConverter: val => toAnyUnit(val, 1, 1, 'ops'),
      },
    ],
  },

  // //////////////////////
  // Storage Panel
  tikv_storage_async_write_duration: {
    title: 'Storage async write duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="write", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99%',
        valConverter: val => timeSecondsFormatter(val, 0),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="write", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_storage_engine_async_request_duration_seconds_sum{type="write", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_storage_engine_async_request_duration_seconds_count{type="write", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'avg',
      },
    ],
  },

  tikv_storage_async_snapshot_duration: {
    title: 'Storage async snapshot duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="snapshot", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99%',
        valConverter: val => timeSecondsFormatter(val, 0),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="snapshot", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_storage_engine_async_request_duration_seconds_sum{type="snapshot", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_storage_engine_async_request_duration_seconds_count{type="write", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'avg',
      },
    ],
  },

  // ///////////////////////
  // Scheduler Panel
  scheduler_pending_commands: {
    title: 'Scheduler pending commands',
    queries: [
      {
        promQLTemplate:
          'sum(tikv_scheduler_contex_total{inspectionid="{{inspectionId}}"}) by (job)',
        labelTemplate: '{{job}}',
      },
    ],
  },

  // ///////////////////////
  // RocksDB - raft Panel
  // write duration
  rocksdb_raft_write_duration: {
    title: 'Write duration',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_write_micro_seconds{db="raft",type="write_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_micro_seconds{db="raft",type="write_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_micro_seconds{db="raft",type="write_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_micro_seconds{db="raft",type="write_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },

  // write stall duration
  rocksdb_raft_write_stall_duration: {
    title: 'Write stall duration',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_write_stall{db="raft",type="write_stall_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_stall{db="raft",type="write_stall_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_stall{db="raft",type="write_stall_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_stall{db="raft",type="write_stall_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },

  // get duration
  rocksdb_raft_get_duration: {
    title: 'Get duration',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_get_micro_seconds{db="raft",type="get_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'avg(tikv_engine_get_micro_seconds{db="raft",type="get_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_get_micro_seconds{db="raft",type="get_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_get_micro_seconds{db="raft",type="get_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },

  // seek duration
  rocksdb_raft_seek_duration: {
    title: 'Seek duration',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_seek_micro_seconds{db="raft",type="seek_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'avg(tikv_engine_seek_micro_seconds{db="raft",type="seek_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_seek_micro_seconds{db="raft",type="seek_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_seek_micro_seconds{db="raft",type="seek_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },

  // wal sync duration
  rocksdb_raft_wal_sync_duration: {
    title: 'WAL sync duration',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_wal_file_sync_micro_seconds{db="raft",type="wal_file_sync_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'avg(tikv_engine_wal_file_sync_micro_seconds{db="raft",type="wal_file_sync_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_wal_file_sync_micro_seconds{db="raft",type="wal_file_sync_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_wal_file_sync_micro_seconds{db="raft",type="wal_file_sync_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },

  // wal sync operation
  rocksdb_raft_wal_sync_operations: {
    title: 'WAL sync operations',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_engine_wal_file_synced{db="raft", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'sync',
        valConverter: val => toAnyUnit(val, 1, 1, 'ops'),
      },
    ],
  },

  rocksdb_raft_number_files_each_level: {
    title: 'Number files at each level',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_num_files_at_level{db="raft", inspectionid="{{inspectionId}}"}) by (cf, level)',
        labelTemplate: 'cf-{{cf}}, level-{{level}}',
        valConverter: toFixed1,
      },
    ],
  },

  rocksdb_raft_compaction_pending_bytes: {
    title: 'Compaction pending bytes',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_engine_pending_compaction_bytes{db="raft", inspectionid="{{inspectionId}}"}[1m])) by (cf)',
        labelTemplate: '{{cf}}',
      },
    ],
  },
  rocksdb_raft_block_cache_size: {
    title: 'Block cache size',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_block_cache_size_bytes{db="raft", inspectionid="{{inspectionId}}"}) by(cf)',
        labelTemplate: '{{cf}}',
        valConverter: bytesSizeFormatter,
      },
    ],
  },

  // ///////////////////////
  // RocksDB - kv Panel
  // write duration
  rocksdb_kv_write_duration: {
    title: 'Write duration',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_write_micro_seconds{db="kv",type="write_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_micro_seconds{db="kv",type="write_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_micro_seconds{db="kv",type="write_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_micro_seconds{db="kv",type="write_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },

  // write stall duration
  rocksdb_kv_write_stall_duration: {
    title: 'Write stall duration',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_write_stall{db="kv",type="write_stall_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_stall{db="kv",type="write_stall_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_stall{db="kv",type="write_stall_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_write_stall{db="kv",type="write_stall_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },
  // get duration
  rocksdb_kv_get_duration: {
    title: 'Get duration',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_get_micro_seconds{db="kv",type="get_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'avg(tikv_engine_get_micro_seconds{db="kv",type="get_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_get_micro_seconds{db="kv",type="get_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_get_micro_seconds{db="kv",type="get_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },

  // seek duration
  rocksdb_kv_seek_duration: {
    title: 'Seek duration',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_seek_micro_seconds{db="kv",type="seek_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'avg(tikv_engine_seek_micro_seconds{db="kv",type="seek_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_seek_micro_seconds{db="kv",type="seek_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_seek_micro_seconds{db="kv",type="seek_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },

  // wal sync duration
  rocksdb_kv_wal_sync_duration: {
    title: 'WAL sync duration',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_wal_file_sync_micro_seconds{db="kv",type="wal_file_sync_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'avg(tikv_engine_wal_file_sync_micro_seconds{db="kv",type="wal_file_sync_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_wal_file_sync_micro_seconds{db="kv",type="wal_file_sync_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'avg(tikv_engine_wal_file_sync_micro_seconds{db="kv",type="wal_file_sync_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },

  // wal sync operation
  rocksdb_kv_wal_sync_operations: {
    title: 'WAL sync operations',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_engine_wal_file_synced{db="kv", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'sync',
        valConverter: val => toAnyUnit(val, 1, 1, 'ops'),
      },
    ],
  },

  rocksdb_kv_number_files_each_level: {
    title: 'Number files at each level',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_num_files_at_level{db="kv", inspectionid="{{inspectionId}}"}) by (cf, level)',
        labelTemplate: 'cf-{{cf}}, level-{{level}}',
        valConverter: toFixed1,
      },
    ],
  },
  rocksdb_kv_compaction_pending_bytes: {
    title: 'Compaction pending bytes',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_engine_pending_compaction_bytes{db="kv", inspectionid="{{inspectionId}}"}[1m])) by (cf)',
        labelTemplate: '{{cf}}',
      },
    ],
  },
  rocksdb_kv_block_cache_size: {
    title: 'Block cache size',
    queries: [
      {
        promQLTemplate:
          'avg(tikv_engine_block_cache_size_bytes{db="kv", inspectionid="{{inspectionId}}"}) by(cf)',
        labelTemplate: '{{cf}}',
        valConverter: bytesSizeFormatter,
      },
    ],
  },
  // ////////////////////
  // Coprocessor Panel
  // request duration
  coprocessor_request_duration: {
    title: 'Request duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.9999, sum(rate(tikv_coprocessor_request_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
        labelTemplate: '{{req}}-99.99%',
        valConverter: val => timeSecondsFormatter(val, 0),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_coprocessor_request_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
        labelTemplate: '{{req}}-99%',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
        labelTemplate: '{{req}}-95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_coprocessor_request_duration_seconds_sum{inspectionid="{{inspectionId}}"}[1m])) by (req) / sum(rate(tikv_coprocessor_request_duration_seconds_count{inspectionid="{{inspectionId}}"}[1m])) by (req)',
        labelTemplate: '{{req}}-avg',
      },
    ],
  },

  // wait duration
  coprocessor_wait_duration: {
    title: 'Wait duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.9999, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
        labelTemplate: '{{req}}-99.99%',
        valConverter: val => timeSecondsFormatter(val, 0),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
        labelTemplate: '{{req}}-99%',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
        labelTemplate: '{{req}}-95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_coprocessor_request_wait_seconds_sum{inspectionid="{{inspectionId}}"}[1m])) by (req) / sum(rate(tikv_coprocessor_request_wait_seconds_count{inspectionid="{{inspectionId}}"}[1m])) by (req)',
        labelTemplate: '{{req}}-avg',
      },
    ],
  },
  // scan keys
  coprocessor_scan_keys: {
    title: 'Scan keys',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.9999, avg(rate(tikv_coprocessor_scan_keys_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, req))',
        labelTemplate: '{{req}}-99.99%',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.99, avg(rate(tikv_coprocessor_scan_keys_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, req))',
        labelTemplate: '{{req}}-99%',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, avg(rate(tikv_coprocessor_scan_keys_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, req))',
        labelTemplate: '{{req}}-95%',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.90, avg(rate(tikv_coprocessor_scan_keys_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, req))',
        labelTemplate: '{{req}}-90%',
      },
    ],
  },

  coprocessor_total_ops_details_table_scan: {
    title: 'Total Ops Details (Table Scan)',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_coprocessor_scan_details{inspectionid="{{inspectionId}}", req="select"}[1m])) by (tag)',
        labelTemplate: '{{tag}}',
      },
    ],
  },
  coprocessor_total_ops_details_index_scan: {
    title: 'Total Ops Details (Index Scan)',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_coprocessor_scan_details{inspectionid="{{inspectionId}}", req="index"}[1m])) by (tag)',
        labelTemplate: '{{tag}}',
      },
    ],
  },

  // 95% Coprocessor wait duration by store
  coprocessor_wait_duration_by_store_95: {
    title: '95% Wait duration by store',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, job,req))',
        labelTemplate: '{{job}}-{{req}}',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
    ],
  },

  // handle_snapshot_duration_99
  handle_snapshot_duration_99: {
    title: '99% Handle snapshot duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_server_send_snapshot_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: 'send',
        valConverter: val => timeSecondsFormatter(val, 1),
      },

      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_raftstore_snapshot_duration_seconds_bucket{type="apply", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: 'apply',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_raftstore_snapshot_duration_seconds_bucket{type="generate", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: 'generate',
      },
    ],
  },

  // thread cpu panel
  raft_store_cpu: {
    title: 'Raft store CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"raftstore_.*", inspectionid="{{inspectionId}}"}[1m])) by (job, name)',
        labelTemplate: '{{job}}',
        valConverter: val => toPercent(val, 2),
      },
    ],
  },

  async_apply_cpu: {
    title: 'Async apply CPU',
    queries: [
      {
        version: 'v_2_x',
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"apply_worker", inspectionid="{{inspectionId}}"}[1m])) by (job, name)',
        labelTemplate: '{{job}}',
        valConverter: val => toPercent(val, 4),
      },
      {
        version: 'v_3_x',
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"apply_[0-9]+", inspectionid="{{inspectionId}}"}[1m])) by (instance)',
        labelTemplate: '{{instance}}',
        valConverter: val => toPercent(val, 2),
      },
    ],
  },

  coprocessor_cpu: {
    title: 'Coprocessor CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"cop_.*", inspectionid="{{inspectionId}}"}[1m])) by (job)',
        labelTemplate: '{{job}}',
        valConverter: val => toPercent(val, 2),
      },
    ],
  },
  storage_readpool_cpu: {
    title: 'Storage ReadPool CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"store_read.*", inspectionid="{{inspectionId}}"}[1m])) by (job)',
        labelTemplate: '{{job}}',
        valConverter: val => toPercent(val, 2),
      },
    ],
  },
  split_check_cpu: {
    title: 'Split check CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"split_check", inspectionid="{{inspectionId}}"}[1m])) by (job)',
        labelTemplate: '{{job}}',
        valConverter: val => toPercent(val, 2),
      },
    ],
  },
  grpc_poll_cpu: {
    title: 'gPRC poll CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"grpc.*", inspectionid="{{inspectionId}}"}[1m])) by (job)',
        labelTemplate: '{{job}}',
        valConverter: val => toPercent(val, 2),
      },
    ],
  },
  scheduler_cpu: {
    title: 'Scheduler CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"storage_schedul.*", inspectionid="{{inspectionId}}"}[1m])) by (job)',
        labelTemplate: '{{job}}',
        valConverter: val => toPercent(val, 2),
      },
    ],
  },

  // gRPC
  grpc_message_duration_99: {
    title: '99% gRPC messge duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_grpc_msg_duration_seconds_bucket{type!="kv_gc", inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
        labelTemplate: '{{type}}',
        valConverter: val => timeSecondsFormatter(val, 0),
      },
    ],
  },

  // 2019/09/18
  tidb_gc_failure_opm: {
    title: 'GC Failure OPM',
    queries: [
      {
        promQLTemplate:
          'sum(increase(tidb_tikvclient_gc_failure{inspectionid="{{inspectionId}}"}[1m])) by (type)',
        labelTemplate: '{{type}}',
      },
    ],
  },
  tidb_gc_delete_range_failure_opm: {
    title: 'Delete Range Failure OPM',
    queries: [
      {
        promQLTemplate:
          'sum(increase(tidb_tikvclient_gc_unsafe_destroy_range_failures{inspectionid="{{inspectionId}}"}[1m])) by (type)',
        labelTemplate: '{{type}}',
      },
    ],
  },

  //= ===============================================================================
  // 重点问题排查
  // cluster-performence-read/write
  // tidb-server
  tidb_server_duration: {
    title: 'Duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tidb_server_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '95',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.8, sum(rate(tidb_server_handle_query_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '80',
      },
    ],
  },
  tidb_server_99_get_token_duration: {
    title: '99% Get Token Duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tidb_server_get_token_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99',
      },
    ],
  },
  tidb_server_connection_count: {
    title: 'Connection Count',
    queries: [
      {
        promQLTemplate: 'tidb_server_connections{inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{instance}}',
      },
      {
        promQLTemplate: 'sum(tidb_server_connections{inspectionid="{{inspectionId}}"})',
        labelTemplate: 'total',
      },
    ],
  },
  tidb_server_heap_memory_usage: {
    title: 'Heap Memory Usage',
    queries: [
      {
        promQLTemplate:
          'go_memstats_heap_inuse_bytes{job=~"tidb.*", inspectionid="{{inspectionId}}"}',
        labelTemplate: '{{instance}}-{{job}}',
      },
    ],
  },
  // parse
  parse_99_parse_duration: {
    title: '99% Parse Duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tidb_session_parse_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, sql_type))',
        labelTemplate: '{{sql_type}}',
      },
    ],
  },
  // compile
  compile_99_compile_duration: {
    title: '99% Compile Duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tidb_session_compile_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, sql_type))',
        labelTemplate: '{{sql_type}}',
      },
    ],
  },
  // transaction
  transaction_duration: {
    title: 'Duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tidb_session_transaction_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, sql_type))',
        labelTemplate: '99-{{sql_type}}',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tidb_session_transaction_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, sql_type))',
        labelTemplate: '95-{{sql_type}}',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.80, sum(rate(tidb_session_transaction_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, sql_type))',
        labelTemplate: '80-{{sql_type}}',
      },
    ],
  },
  transaction_statement_num: {
    title: 'Transaction Statement Num',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tidb_session_transaction_statement_num_bucket{inspectionid="{{inspectionId}}"}[30s])) by (le, sql_type))',
        labelTemplate: '99-{{sql_type}}',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.80, sum(rate(tidb_session_transaction_statement_num_bucket{inspectionid="{{inspectionId}}"}[30s])) by (le, sql_type))',
        labelTemplate: '80-{{sql_type}}',
      },
    ],
  },
  transaction_retry_num: {
    title: 'Transaction Retry Num',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(1.0, sum(rate(tidb_session_retry_num_bucket{inspectionid="{{inspectionId}}"}[30s])) by (le))',
        labelTemplate: '100',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tidb_session_retry_num_bucket{inspectionid="{{inspectionId}}"}[30s])) by (le))',
        labelTemplate: '99',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.90, sum(rate(tidb_session_retry_num_bucket{inspectionid="{{inspectionId}}"}[30s])) by (le))',
        labelTemplate: '90',
      },
    ],
  },
  // kv
  kv_cmd_duration_9999: {
    title: 'KV Cmd Duration 9999',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.999, sum(rate(tidb_tikvclient_txn_cmd_duration_seconds_bucket{type=~"get|batch_get|seek|seek_reverse", inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
        labelTemplate: '{{type}}',
      },
    ],
  },
  kv_cmd_duration_99: {
    title: 'KV Cmd Duration 99',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tidb_tikvclient_txn_cmd_duration_seconds_bucket{type=~"get|batch_get|seek|seek_reverse", inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
        labelTemplate: '{{type}}',
      },
    ],
  },
  kv_lock_resolve_ops: {
    title: 'Lock Resolve OPS',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tidb_tikvclient_lock_resolver_actions_total{inspectionid="{{inspectionId}}"}[1m])) by (type)',
        labelTemplate: '{{type}}',
      },
    ],
  },
  kv_99_kv_backoff_duration: {
    title: '99% KV Backoff Duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tidb_tikvclient_backoff_seconds_bucket{inspectionid="{{inspectionId}}"}[5m])) by (le, type))',
        labelTemplate: '{{type}}',
      },
    ],
  },
  kv_backoff_ops: {
    title: 'KV Backoff OPS',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tidb_tikvclient_backoff_total{inspectionid="{{inspectionId}}"}[1m])) by (type)',
        labelTemplate: '{{type}}',
      },
    ],
  },
  // PD Client
  pd_tso_wait_duration: {
    title: 'PD TSO Wait Duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.999, sum(rate(pd_client_cmd_handle_cmds_duration_seconds_bucket{type="tso", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '999',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(pd_client_cmd_handle_cmds_duration_seconds_bucket{type="tso", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '99',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.90, sum(rate(pd_client_cmd_handle_cmds_duration_seconds_bucket{type="tso", inspectionid="{{inspectionId}}"}[1m])) by (le))',
        labelTemplate: '90',
      },
    ],
  },
  // pd_tso_rpc_duration: existed above
  // gRPC
  grpc_99_grpc_message_duration: {
    title: '99% gRPC messge duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_grpc_msg_duration_seconds_bucket{type=~"kv_get|kv_batch_get|coprocessor", inspectionid="{{inspectionId}}"}[5m])) by (le, type))',
        labelTemplate: '{{type}}',
      },
    ],
  },
  grpc_poll_cpu_2: {
    title: 'gRPC poll CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"raftstore_.*", inspectionid="{{inspectionId}}"}[1m])) by (instance, name)',
        labelTemplate: '{{instance}} - {{name}}',
      },
    ],
  },
  // Disk
  disk_latency: {
    title: 'Disk Latency',
    queries: [
      {
        promQLTemplate:
          'irate(node_disk_read_time_seconds_total{inspectionid="{{inspectionId}}"}[5m]) / irate(node_disk_reads_completed_total{inspectionid="{{inspectionId}}"}[5m])',
        labelTemplate: 'Read: {{instance}} - {{device}}',
      },
    ],
  },
  disk_operations: {
    title: 'Disk Operations',
    queries: [
      {
        promQLTemplate:
          'irate(node_disk_reads_completed_total{inspectionid="{{inspectionId}}"}[5m])',
        labelTemplate: 'Read: {{instance}} - {{device}}',
      },
    ],
  },
  disk_bandwidth: {
    title: 'Disk Bandwidth',
    queries: [
      {
        promQLTemplate: 'irate(node_disk_read_bytes_total{inspectionid="{{inspectionId}}"}[5m])',
        labelTemplate: 'Read: {{instance}} - {{device}}',
      },
    ],
  },
  disk_load: {
    title: 'Disk Load',
    queries: [
      {
        promQLTemplate:
          'irate(node_disk_read_time_seconds_total{inspectionid="{{inspectionId}}"}[5m])',
        labelTemplate: 'Read: {{instance}} - {{device}}',
      },
    ],
  },
  // Storage
  storage_readpool_cpu_2: {
    title: 'Storage ReadPool CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"store_read.*", inspectionid="{{inspectionId}}"}[1m])) by (instance)',
        labelTemplate: '{{instance}}',
      },
    ],
  },
  // Coprocessor
  coprocessor_wait_duration_2: {
    title: 'Wait duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(1, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
        labelTemplate: '{{req}}-100%',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_coprocessor_request_wait_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
        labelTemplate: '{{req}}-99%',
      },
    ],
  },
  coprocessor_handle_duration: {
    title: 'Handle duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(1, sum(rate(tikv_coprocessor_request_handle_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
        labelTemplate: '{{req}}-100%',
      },
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_coprocessor_request_handle_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le,req))',
        labelTemplate: '{{req}}-99%',
      },
    ],
  },
  coprocessor_cpu_2: {
    title: 'Coprocessor CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"cop_.*", inspectionid="{{inspectionId}}"}[1m])) by (instance)',
        labelTemplate: '{{instance}}',
      },
    ],
  },
  // RocksDB-KV
  // rocksdb_kv_get_duration: existed above
  rocksdb_kv_get_operation: {
    title: 'Get operations',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_engine_memtable_efficiency{db="kv", type="memtable_hit", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'memtable',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_cache_efficiency{db="kv", type=~"block_cache_data_hit|block_cache_filter_hit", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'block_cache',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_get_served{db="kv", type="get_hit_l0", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'l0',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_get_served{db="kv", type="get_hit_l1", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'l1',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_get_served{db="kv", type="get_hit_l2_and_up", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'l2_and_up',
      },
    ],
  },
  // rocksdb_kv_seek_duration: existed above
  rocksdb_kv_seek_operation: {
    title: 'Seek operations',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_engine_locate{db="kv", type="number_db_seek", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'seek',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_locate{db="kv", type="number_db_seek_found", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'seek_found',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_locate{db="kv", type="number_db_next", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'next',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_locate{db="kv", type="number_db_next_found", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'next_found',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_locate{db="kv", type="number_db_prev", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'prev',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_locate{db="kv", type="number_db_prev_found", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'prev_found',
      },
    ],
  },
  rocksdb_kv_block_cache_hit: {
    title: 'Block Cache hit',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_engine_cache_efficiency{instance=~"$instance", db="$db", type="block_cache_hit", inspectionid="{{inspectionId}}"}[1m])) / (sum(rate(tikv_engine_cache_efficiency{db="$db", type="block_cache_hit", inspectionid="{{inspectionId}}"}[1m])) + sum(rate(tikv_engine_cache_efficiency{db="$db", type="block_cache_miss", inspectionid="{{inspectionId}}"}[1m])))',
        labelTemplate: 'all',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_cache_efficiency{db="kv", type="block_cache_data_hit", inspectionid="{{inspectionId}}"}[1m])) / (sum(rate(tikv_engine_cache_efficiency{db="kv", type="block_cache_data_hit", inspectionid="{{inspectionId}}"}[1m])) + sum(rate(tikv_engine_cache_efficiency{db="kv", type="block_cache_data_miss", inspectionid="{{inspectionId}}"}[1m])))',
        labelTemplate: 'data',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_cache_efficiency{db="kv", type="block_cache_filter_hit", inspectionid="{{inspectionId}}"}[1m])) / (sum(rate(tikv_engine_cache_efficiency{db="kv", type="block_cache_filter_hit", inspectionid="{{inspectionId}}"}[1m])) + sum(rate(tikv_engine_cache_efficiency{db="kv", type="block_cache_filter_miss", inspectionid="{{inspectionId}}"}[1m])))',
        labelTemplate: 'filter',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_cache_efficiency{db="kv", type="block_cache_index_hit", inspectionid="{{inspectionId}}"}[1m])) / (sum(rate(tikv_engine_cache_efficiency{db="kv", type="block_cache_index_hit", inspectionid="{{inspectionId}}"}[1m])) + sum(rate(tikv_engine_cache_efficiency{db="kv", type="block_cache_index_miss", inspectionid="{{inspectionId}}"}[1m])))',
        labelTemplate: 'index',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_engine_bloom_efficiency{db="kv", type="bloom_prefix_useful", inspectionid="{{inspectionId}}"}[1m])) / sum(rate(tikv_engine_bloom_efficiency{db="kv", type="bloom_prefix_checked", inspectionid="{{inspectionId}}"}[1m]))',
        labelTemplate: 'bloom prefix',
      },
    ],
  },
  //-------------------------
  // scheduler
  scheduler_latch_wait_duration: {
    title: 'Scheduler latch wait duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[5m])) by (le, type))',
        labelTemplate: '99%-{{type}}',
        valConverter: val => timeSecondsFormatter(val, 1),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_scheduler_latch_wait_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[5m])) by (le, type))',
        labelTemplate: '95%-{{type}}',
      },
      {
        promQLTemplate:
          'rate(tikv_scheduler_latch_wait_duration_seconds_sum{inspectionid="{{inspectionId}}"}[5m]) / rate(tikv_scheduler_latch_wait_duration_seconds_count{inspectionid="{{inspectionId}}"}[5m])',
        labelTemplate: 'avg-{{type}}',
      },
    ],
  },
  scheduler_worker_cpu: {
    title: 'Scheduler worker CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"sched_.*", inspectionid="{{inspectionId}}"}[1m])) by (instance)',
        labelTemplate: '{{instance}}',
      },
    ],
  },
  scheduler_99_command_duration: {
    title: '99% scheduler command duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_scheduler_command_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, type))',
        labelTemplate: '{{type}}',
      },
    ],
  },
  // raftstore
  raftstore_99_propose_wait_duration: {
    title: '99% propose wait duration by instance',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_raftstore_request_wait_time_duration_secs_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, instance))',
        labelTemplate: '{{instance}}',
      },
    ],
  },
  raftstore_raft_store_cpu: {
    title: 'Raft store CPU',
    queries: [
      {
        promQLTemplate:
          'sum(rate(tikv_thread_cpu_seconds_total{name=~"raftstore_.*", inspectionid="{{inspectionId}}"}[1m])) by (instance)',
        labelTemplate: '{{instance}}',
      },
    ],
  },
  raftstore_storage_async_write_duration: {
    title: 'Storage async write duration',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="write", inspectionid="{{inspectionId}}"}[5m])) by (le))',
        labelTemplate: '99%',
        valConverter: val => timeSecondsFormatter(val, 0),
      },
      {
        promQLTemplate:
          'histogram_quantile(0.95, sum(rate(tikv_storage_engine_async_request_duration_seconds_bucket{type="write", inspectionid="{{inspectionId}}"}[5m])) by (le))',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'sum(rate(tikv_storage_engine_async_request_duration_seconds_sum{type="write", inspectionid="{{inspectionId}}"}[5m])) / sum(rate(tikv_storage_engine_async_request_duration_seconds_count{type="write", inspectionid="{{inspectionId}}"}[5m]))',
        labelTemplate: 'avg',
      },
    ],
  },
  // RocksDB-Raft
  rocksdb_raft_99_append_log_duration: {
    title: '99% append log duration by instance',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_raftstore_append_log_duration_seconds_bucket{inspectionid="{{inspectionId}}"}[1m])) by (le, instance))',
        labelTemplate: '{{instance}}',
      },
    ],
  },
  rocksdb_raft_write_duration_2: {
    title: 'Write duration',
    queries: [
      {
        promQLTemplate:
          'max(tikv_engine_write_micro_seconds{db="raft",type="write_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'max(tikv_engine_write_micro_seconds{db="raft",type="write_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'max(tikv_engine_write_micro_seconds{db="raft",type="write_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'max(tikv_engine_write_micro_seconds{db="raft",type="write_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },
  // RocksDB-KV
  rocksdb_kv_99_apply_wait_duration: {
    title: '99% apply wait duration by instance',
    queries: [
      {
        promQLTemplate:
          'histogram_quantile(0.99, sum(rate(tikv_raftstore_apply_wait_time_duration_secs_bucket{inspectionid="{{inspectionId}}"}[5m])) by (le, instance))',
        labelTemplate: '{{instance}}',
      },
    ],
  },
  // rocksdb_kv_apply_log_duration: same as tikv_apply_log_duration
  rocksdb_kv_write_duration_2: {
    title: 'Write duration',
    queries: [
      {
        promQLTemplate:
          'max(tikv_engine_write_micro_seconds{db="kv",type="write_max", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'max',
        valConverter: val => timeSecondsFormatter(val / (1000 * 1000), 1),
      },
      {
        promQLTemplate:
          'max(tikv_engine_write_micro_seconds{db="kv",type="write_percentile99", inspectionid="{{inspectionId}}"})',
        labelTemplate: '99%',
      },
      {
        promQLTemplate:
          'max(tikv_engine_write_micro_seconds{db="kv",type="write_percentile95", inspectionid="{{inspectionId}}"})',
        labelTemplate: '95%',
      },
      {
        promQLTemplate:
          'max(tikv_engine_write_micro_seconds{db="kv",type="write_average", inspectionid="{{inspectionId}}"})',
        labelTemplate: 'avg',
      },
    ],
  },
  // rocksdb_kv_async_apply_cpu: same as async_apply_cpu
};
