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
    charts: ['memory_total'],
  },
  global_cpu_usage: {
    title: 'Global CPU Usage',
    expand: true,
    charts: ['global_cpu_usage'],
  },
  global_load: {
    title: 'Load',
    expand: true,
    charts: ['load_1'],
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
    charts: ['qps', 'qps_by_instance', 'tidb_duration', 'failed_query_opm', 'slow_query'],
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
    charts: ['pd_client_cmd_fail_ops', 'pd_tso_rpc_duration'],
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
      'tikv_thread_cpu',
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

  // 新的分类
  dba_error_1: {
    title: 'Error 信息 (一级)',
    charts: [
      'failed_query_opm',
      // 'Critical Error OPS', // todo
      'pd_client_cmd_fail_ops',
      // 'panic count' // todo
    ],
  },
  dba_error_2: {
    title: 'Error 信息 (二级)',
    charts: [
      // '服务器时间回跳' // todo
      'ticlient_region_error',
      'lock_resolve_ops',
      'tikv_server_busy',
      'tikv_raftstore_error',
      'tikv_coprocessor_error',
      'tikv_grpc_message_error',
      'tikv_leader_drop',
      'tikv_leader_missing',
      'tikv_scheduler_error',
      'tikv_server_report_failures',
      'schema_lease_error_opm',
      'tidb_gc_failure_opm',
      'tidb_gc_delete_range_failure_opm',
      // 'tikv GC 进度' // todo
    ],
  },
  dba_resources: {
    title: '资源信息',
    charts: [
      'vcores',
      'uptime',

      // 'global_cpu_usage',
      'pd_cpu_usage',
      'tidb_cpu_usage',
      'tikv_cpu_usage',
      'tikv_thread_cpu',
      'raft_store_cpu',
      // 'async_apply_cpu',
      'coprocessor_cpu',
      // 'storage_readpool_cpu',
      // 'split_check_cpu',
      // 'grpc_poll_cpu',
      // 'scheduler_cpu',

      // memorey // todo
      'memory_total',
      'memory_available',

      'load_1',
      'load_5',
      'load_15',

      'storage_capacity',
      'storage_size',
      'storage_size_ratio',
      'io_util',
      // 'IO iops' // todo
      // 'IO write latency'  // todo

      // network
      'network_traffic',
      'connection_count',
      'tcp_retrans',

      // store/node // todo

      // region
      // 'total' // todo
      'regions_label_level',
      'region_health',
      'region_heartbeat_latency_99',

      // qps - tidb
      'qps',
      'tidb_duration',

      // qps - pd
      // qps // todo
      'handle_request_duration_seconds',
      'pd_tso_rpc_duration',
      // async duration // todo

      // etcd
      'handle_txn_count',
      'wal_fsync_duration_seconds_99',

      // qps - tikv
      'tikv_qps',
      // grpc duration

      // ddl
      'ddl_opm',

      // tidb get token duraion // todo
    ],
  },
  dba_background_task_info: {
    title: '后台任务信息',
    charts: [
      'load_schema_duration',
      'schema_lease_error_opm',

      // scheduler
      'tikv_scheduler_prewrite_latch_wait_duration',
      'tivk_scheduler_prewrite_command_duration',
      'tikv_scheduler_commit_latch_wait_duration',
      'tivk_scheduler_commit_command_duration',
      'scheduler_pending_commands',

      'schedule_operator_create',
      'schedule_operator_timeout',

      // snapshot
      'handle_snapshot_duration_99',
      // Snapshot state count // todo

      // balance // todo

      // raft
      'tikv_raft_vote',
    ],
  },

  // 重点问题排查相关 panel
  tidb_server_2: {
    title: 'TiDB-Server',
    charts: [
      'tidb_server_duration',
      'tidb_server_99_get_token_duration',
      'tidb_server_connection_count',
      'tidb_server_heap_memory_usage',
    ],
  },
  parse: {
    title: 'Parse',
    charts: ['parse_99_parse_duration'],
  },
  compile: {
    title: 'Compile',
    charts: ['compile_99_compile_duration'],
  },
  transaction: {
    title: 'Transaction',
    charts: ['transaction_duration', 'transaction_statement_num', 'transaction_retry_num'],
  },
  kv: {
    title: 'KV',
    charts: [
      'kv_cmd_duration_9999',
      'kv_cmd_duration_99',
      'kv_lock_resolve_ops',
      'kv_99_kv_backoff_duration',
      'kv_backoff_ops',
    ],
  },
  pd_client: {
    title: 'PD Client',
    charts: ['pd_tso_wait_duration', 'pd_tso_rpc_duration'],
  },
  grpc: {
    title: 'gRPC',
    charts: ['grpc_99_grpc_message_duration', 'grpc_poll_cpu_2'],
  },
  disk: {
    title: 'Disk',
    charts: ['disk_latency', 'disk_operations', 'disk_bandwidth', 'disk_load'],
  },
  //---------
  // only for read
  storage: {
    title: 'Storage (only for read)',
    charts: ['storage_readpool_cpu_2'],
  },
  coprocessor: {
    title: 'Coprocessor (only for read)',
    charts: ['coprocessor_wait_duration_2', 'coprocessor_handle_duration', 'coprocessor_cpu_2'],
  },
  rocksdb_kv: {
    title: 'RocksDB-KV (only for read)',
    charts: [
      'rocksdb_kv_get_duration',
      'rocksdb_kv_get_operation',
      'rocksdb_kv_seek_duration',
      'rocksdb_kv_seek_operation',
      'rocksdb_kv_block_cache_hit',
    ],
  },
  //---------
  // only for write
  scheduler: {
    title: 'Scheduler (only for write)',
    charts: [
      'scheduler_latch_wait_duration',
      'scheduler_worker_cpu',
      'scheduler_99_command_duration',
    ],
  },
  raftstore: {
    title: 'raftstore (only for write)',
    charts: [
      'raftstore_99_propose_wait_duration',
      'raftstore_raft_store_cpu',
      'raftstore_storage_async_write_duration',
    ],
  },
  rocksdb_raft: {
    title: 'RocksDB-Raft (only for write)',
    charts: ['rocksdb_raft_99_append_log_duration', 'rocksdb_raft_write_duration_2'],
  },
  rocksdb_kv_2: {
    title: 'RocksDB-KV (only for write)',
    charts: [
      'rocksdb_kv_99_apply_wait_duration',
      'tikv_apply_log_duration',
      'rocksdb_kv_write_duration_2',
      'async_apply_cpu',
    ],
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

export const DBA_PANELS = [
  'dba_error_1',
  'dba_error_2',
  'dba_resources',
  'dba_background_task_info',
];

export const TROUBLE_SHOOTING_PANELS = [];

export const NODE_STORE_INFO_PANELS = [];

export const EMPHASIS_DB_PERFORMANCE_PANELS = [
  'tidb_server_2',
  'parse',
  'compile',
  'transaction',
  'kv',
  'pd_client',
  'grpc',
  'disk',
  //---------------------------
  // only for read
  'storage',
  'coprocessor',
  'rocksdb_kv',
  //---------------------------
  // only for write
  'scheduler',
  'raftstore',
  'rocksdb_raft',
  'rocksdb_kv_2',
];
