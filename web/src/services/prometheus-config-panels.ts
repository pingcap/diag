export interface IPanel {
  title: string;
  expand?: boolean; // default is false
  charts: string[];
}

export const ALL_PANELS: { [key: string]: IPanel } = {
  // global
  global_vcores: {
    title: 'Vcores',
    expand: true,
    charts: ['vcores'],
  },
  global_memory: {
    title: 'Memory',
    expand: true,
    charts: ['memory'],
  },
  global_cpu_usage: {
    title: 'CPU Usage',
    expand: true,
    charts: ['cpu_usage'],
  },
  global_load: {
    title: 'Load',
    expand: true,
    charts: ['load'],
  },
  global_memory_available: {
    title: 'Memorey Available',
    expand: true,
    charts: ['memory_available'],
  },
  global_network_traffic: {
    title: 'Network Traffic',
    expand: true,
    charts: ['network_traffic'],
  },
  global_tcp_retrans: {
    title: 'TCP Retrans',
    expand: true,
    charts: ['tcp_retrans'],
  },
  global_io_util: {
    title: 'IO Util',
    expand: true,
    charts: ['io_util'],
  },

  // PD Panel
  pd_cluster: {
    title: 'Cluster',
    charts: [
      'stores_status',
      'storage_capacity',
      'storage_size',
      'storage_size_ratio',
      'regions_label_level',
      'region_health',
    ],
  },
  pd_balance: {
    title: 'Balance',
    charts: [
      'store_available',
      'store_available_ratio',
      'store_leader_score',
      'store_region_score',
      'store_leader_count',
    ],
  },
  pd_hot_region: {
    title: 'Hot Region',
    charts: [
      'hot_write_region_leader_distribution',
      'hot_write_region_peer_distribution',
      'hot_read_region_leader_distribution',
    ],
  },
  pd_operator: {
    title: 'Operator',
    charts: ['schedule_operator_create', 'schedule_operator_timeout'],
  },
  pd_etcd: {
    title: 'Etcd',
    charts: ['handle_txn_count', 'wal_fsync_duration_seconds_99'],
  },
  pd_tidb: {
    title: 'TiDB',
    charts: ['handle_request_duration_seconds'],
  },
  pd_heartbeat: {
    title: 'Heartbeat',
    charts: ['region_heartbeat_latency_99'],
  },
  pd_grpc: {
    title: 'gRPC',
    charts: ['grpc_completed_commands_duration_99'],
  },

  // TiDB Panel
  tidb_query_summary: {
    title: 'Query Summary',
    charts: ['qps', 'qps_by_instance', 'duration', 'failed_query_opm', 'slow_query'],
  },
  tidb_server: {
    title: 'Server',
    charts: [
      'uptime',
      'tidb_cpu_usage',
      'connection_count',
      'goroutine_count',
      'heap_memory_usage',
    ],
  },
  tidb_distsql: {
    title: 'Distsql',
    charts: ['distsql_duration'],
  },
  tidb_kv_errors: {
    title: 'KV Errors',
    charts: ['ticlient_region_error', 'lock_resolve_ops'],
  },
  tidb_pd_client: {
    title: 'PD Client',
    charts: ['pod_client_cmd_fail_ops', 'pd_tso_rpc_duration'],
  },
  tidb_schema_load: {
    title: 'Schema Load',
    charts: ['load_schema_duration', 'schema_lease_error_opm'],
  },
  tidb_ddl: {
    title: 'DDL',
    charts: ['ddl_opm'],
  },

  // TiKV Panel
  tikv_cluster: {
    title: 'Cluster',
    charts: [
      'tikv_store_size',
      'tikv_cpu',
      'tikv_memory',
      'tikv_io_utilization',
      'tikv_qps',
      'tikv_leader',
    ],
  },
  tikv_errors: {
    title: 'Errors',
    charts: [
      'tikv_server_busy',
      'tikv_server_report_failures',
      'tikv_raftstore_error',
      'tikv_scheduler_error',
      'tikv_coprocessor_error',
      'tikv_grpc_message_error',
      'tikv_leader_drop',
      'tikv_leader_missing',
    ],
  },
  tikv_server: {
    title: 'Server',
    charts: ['tikv_channel_full', 'tikv_approximate_region_size'],
  },
  tikv_raft_io: {
    title: 'Raft IO',
    charts: [
      'tikv_apply_log_duration',
      'tikv_apply_log_duration_per_server',
      'tikv_append_log_duration',
      'tikv_append_log_duration_per_server',
    ],
  },
  tikv_scheduler_prewrite: {
    title: 'Scheduler - prewrite',
    charts: [
      'tikv_scheduler_prewrite_latch_wait_duration',
      'tivk_scheduler_prewrite_command_duration',
    ],
  },
  tikv_scheduler_commit: {
    title: 'Scheduler - commit',
    charts: ['tikv_scheduler_commit_latch_wait_duration', 'tivk_scheduler_commit_command_duration'],
  },
  tikv_raft_propose: {
    title: 'Raft propose',
    charts: ['tikv_propose_wait_duration'],
  },
  tikv_raft_message: {
    title: 'Raft message',
    charts: ['tikv_raft_vote'],
  },
  tikv_storage: {
    title: 'Storage',
    charts: ['tikv_storage_async_write_duration', 'tikv_storage_async_snapshot_duration'],
  },
  tikv_scheduler: {
    title: 'Scheduler',
    charts: ['scheduler_pending_commands'],
  },
  tikv_rocks_db_raft: {
    title: 'RocksDB - raft',
    charts: [
      'rocksdb_raft_write_duration',
      'rocksdb_raft_write_stall_duration',
      'rocksdb_raft_get_duration',
      'rocksdb_raft_seek_duration',
      'rocksdb_raft_wal_sync_duration',
      'rocksdb_raft_wal_sync_operations',
      'rocksdb_raft_number_files_each_level',
      'rocksdb_raft_compaction_pending_bytes',
      'rocksdb_raft_block_cache_size',
    ],
  },
  tikv_rocksdb_kv: {
    title: 'RocksDB - kv',
    charts: [
      'rocksdb_kv_write_duration',
      'rocksdb_kv_write_stall_duration',
      'rocksdb_kv_get_duration',
      'rocksdb_kv_seek_duration',
      'rocksdb_kv_wal_sync_duration',
      'rocksdb_kv_wal_sync_operations',
      'rocksdb_kv_number_files_each_level',
      'rocksdb_kv_compaction_pending_bytes',
      'rocksdb_kv_block_cache_size',
    ],
  },
  tikv_coprocessor: {
    title: 'Coprocessor',
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
    charts: ['handle_snapshot_duration_99'],
  },
  tikv_thread_cpu: {
    title: 'Thread CPU',
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
    charts: ['grpc_message_duration_99'],
  },
};

// /////////////////////////////////////////

export const GLOBAL_PANNELS = [
  'global_vcores',
  'global_memory',
  'global_cpu_usage',
  'global_load',
  'global_memory_available',
  'global_network_traffic',
  'global_tcp_retrans',
  'global_io_util',
];

export const PD_PANELS = [
  'pd_cluster',
  'pd_balance',
  'pd_hot_region',
  'pd_operator',
  'pd_etcd',
  'pd_tidb',
  'pd_heartbeat',
  'pd_grpc',
];

export const TIDB_PANELS = [
  'tidb_query_summary',
  'tidb_server',
  'tidb_distsql',
  'tidb_kv_errors',
  'tidb_pd_client',
  'tidb_schema_load',
  'tidb_ddl',
];

export const TIKV_PANELS = [
  'tikv_cluster',
  'tikv_errors',
  'tikv_server',
  'tikv_raft_io',
  'tikv_scheduler_prewrite',
  'tikv_scheduler_commit',
  'tikv_raft_propose',
  'tikv_raft_message',
  'tikv_storage',
  'tikv_scheduler',
  'tikv_rocks_db_raft',
  'tikv_rocksdb_kv',
  'tikv_coprocessor',
  'tikv_snapshot',
  'tikv_thread_cpu',
  'tikv_grpc',
];
