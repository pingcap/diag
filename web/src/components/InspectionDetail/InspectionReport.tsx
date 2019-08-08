import React from 'react';
import moment from 'moment';
import { IInspectionDetail } from '@/models/inspection';
import AutoTable from './AutoTable';
import AutoObjectTable from './AutoObjectTable';
import {
  fillPromQLTemplate,
  RAW_METRICS_ARR,
  IPromParams,
  IMetric,
  IRawMetric,
} from '@/services/prometheus';
import CollpasePanel from './CollapsePanel';
import PrometheusChart from './PrometheusChart';

interface InspectionReportProps {
  inspection: IInspectionDetail;
}

const CHART_SAMPLE_COUNT = 15;

function InspectionReport({ inspection }: InspectionReportProps) {
  const { report } = inspection;

  const start = moment(inspection.scrape_begin).unix();
  const end = moment(inspection.scrape_end).unix();
  const step = Math.floor((end - start) / CHART_SAMPLE_COUNT);
  const promParams: IPromParams = { start, end, step };

  function renderPromethuesChart(rawMetricKey: string, title?: string) {
    const rawMetrics: IRawMetric[] = RAW_METRICS_ARR[rawMetricKey];
    const metrics: IMetric[] = fillPromQLTemplate(rawMetrics, inspection.uuid);
    const finalTitle = title || metrics[0].title;
    return <PrometheusChart title={finalTitle} promMetrics={metrics} promParams={promParams} />;
  }

  return (
    <div style={{ marginTop: 20 }}>
      {/*
      <CollpasePanel title="" expand={true}>
        {renderPromethuesChart('')}
      </CollpasePanel>
       */}

      <h2>一、全局诊断</h2>
      <AutoTable title="overview" dataArr={report.symptoms} />

      <h2>二、基本信息</h2>
      <AutoObjectTable title="1、基本信息" data={report.basic || {}} />
      <AutoTable title="2、数据库基本信息" dataArr={report.dbinfo || []} expand={false} />
      <AutoTable title="3、资源信息" dataArr={report.resource || []} />
      <AutoTable title="4、告警信息" dataArr={report.alert || []} />
      <AutoTable title="5、慢查询信息" dataArr={report.slow_log || []} />
      <AutoTable title="6、硬件信息" dataArr={report.hardware || []} />
      <AutoTable title="7、软件信息" dataArr={report.software_version || []} />
      <AutoTable title="8、软件配置信息" dataArr={report.software_config || []} expand={false} />
      <AutoTable title="9、机器 NTP 时间同步信息" dataArr={[]} />
      <AutoTable title="10、网络质量信息" dataArr={report.network || []} />
      <AutoTable title="11、集群拓扑结构信息" dataArr={[]} />
      <AutoTable title="12、demsg 信息" dataArr={report.demsg || []} expand={false} />

      <h2>三、监控信息</h2>
      <h3>1、全局监控</h3>
      <CollpasePanel title="Vcores">{renderPromethuesChart('vcores')}</CollpasePanel>
      <CollpasePanel title="Memory">{renderPromethuesChart('memory')}</CollpasePanel>
      <CollpasePanel title="CPU Usage">{renderPromethuesChart('cpu_usage')}</CollpasePanel>
      <CollpasePanel title="Load">{renderPromethuesChart('load')}</CollpasePanel>
      <CollpasePanel title="Memorey Available">
        {renderPromethuesChart('memory_available')}
      </CollpasePanel>
      <CollpasePanel title="Network Traffic">
        {renderPromethuesChart('network_traffic')}
      </CollpasePanel>
      <CollpasePanel title="TCP Retrans">{renderPromethuesChart('tcp_retrans')}</CollpasePanel>
      <CollpasePanel title="IO Util">{renderPromethuesChart('io_util')}</CollpasePanel>

      <h3>2、PD</h3>
      <CollpasePanel title="Cluster" expand={false}>
        {renderPromethuesChart('stores_status', 'Store Status')}
        {renderPromethuesChart('storage_capacity', 'Store Capacity')}
        {renderPromethuesChart('storage_size', 'Storage Size')}
        {renderPromethuesChart('storage_size_ratio', 'Storage Size Ratio')}
        {renderPromethuesChart('regions_label_level', 'Region Label Isolation Level')}
        {renderPromethuesChart('region_health', 'Region Health')}
      </CollpasePanel>
      <CollpasePanel title="Balance" expand={false}>
        {renderPromethuesChart('store_available', 'Store Available')}
        {renderPromethuesChart('store_leader_score', 'Store Leader Score')}
        {renderPromethuesChart('store_region_score', 'Store Region Score')}
        {renderPromethuesChart('store_leader_count', 'Store Leader Count')}
      </CollpasePanel>
      <CollpasePanel title="Hot Region" expand={false}>
        {renderPromethuesChart(
          'hot_write_region_leader_distribution',
          "Hot write region's leader distribution",
        )}
        {renderPromethuesChart(
          'hot_write_region_peer_distribution',
          "Hot write region's peer distribution",
        )}
        {renderPromethuesChart(
          'hot_read_region_leader_distribution',
          "Hot read region's leader distribution",
        )}
      </CollpasePanel>
      <CollpasePanel title="Operator" expand={false}>
        {renderPromethuesChart('schedule_operator_create', 'Schedule Operator Create')}
        {renderPromethuesChart('schedule_operator_timeout', 'Schedule Operator Timeout')}
      </CollpasePanel>
      <CollpasePanel title="Etcd" expand={false}>
        {renderPromethuesChart('handle_txn_count', 'handle txn count')}
        {renderPromethuesChart('wal_fsync_duration_seconds_99', '99% wal fsync duration seconds')}
      </CollpasePanel>
      <CollpasePanel title="TiDB" expand={false}>
        {renderPromethuesChart(
          'handle_request_duration_seconds',
          'handle request duration seconds',
        )}
      </CollpasePanel>
      <CollpasePanel title="Heartbeat" expand={false}>
        {renderPromethuesChart('region_heartbeat_latency_99', '99% region heartbeat latency')}
      </CollpasePanel>

      <h3>3、TiDB</h3>
      <CollpasePanel title="Query Summary" expand={false}>
        {renderPromethuesChart('qps', 'QPS')}
        {renderPromethuesChart('qps_by_instance', 'QPS By Instance')}
        {renderPromethuesChart('duration', 'Duration')}
        {renderPromethuesChart('failed_query_opm', 'Failed Query OPM')}
      </CollpasePanel>
      <CollpasePanel title="Server" expand={false}>
        {renderPromethuesChart('connection_count_all', 'Connection Count')}
        {renderPromethuesChart('goroutine_count', 'Goroutine Count')}
        {renderPromethuesChart('heap_memory_usage', 'Heap Memory Usage')}
      </CollpasePanel>
      <CollpasePanel title="Distsql" expand={false}>
        {renderPromethuesChart('distsql_duration', 'Distsql Duration')}
      </CollpasePanel>
      <CollpasePanel title="KV Errors" expand={false}>
        {renderPromethuesChart('ticlient_region_error', 'TiClient Region Error OPS')}
        {renderPromethuesChart('lock_resolve_ops', 'Lock Resolve OPS')}
      </CollpasePanel>
      <CollpasePanel title="PD Client" expand={false}>
        {renderPromethuesChart('pod_client_cmd_fail_ops', 'PD Client CMD Fail OPS')}
        {renderPromethuesChart('pd_tso_rpc_duration', 'PD TSO RPC Duration')}
      </CollpasePanel>
      <CollpasePanel title="Schema Load" expand={false}>
        {renderPromethuesChart('load_schema_duration', 'Load Schema Duration')}
        {renderPromethuesChart('schema_lease_error_opm', 'Schema Lease Error OPM')}
      </CollpasePanel>
      <CollpasePanel title="DDL" expand={false}>
        {renderPromethuesChart('ddl_opm', 'DDL OPM')}
      </CollpasePanel>

      <h3>4、TiKV</h3>
      <CollpasePanel title="Cluster" expand={false}>
        {renderPromethuesChart('tikv_store_size', 'Store Size')}
        {renderPromethuesChart('tikv_cpu', 'CPU')}
        {renderPromethuesChart('tikv_memory', 'Memory')}
        {renderPromethuesChart('tikv_io_utilization', 'IO Utilization')}
        {renderPromethuesChart('tikv_qps')}
        {renderPromethuesChart('tikv_leader')}
      </CollpasePanel>
      <CollpasePanel title="Errors" expand={false}>
        {renderPromethuesChart('tikv_server_busy')}
        {renderPromethuesChart('tikv_server_report_failures')}
        {renderPromethuesChart('tikv_raftstore_error')}
        {renderPromethuesChart('tikv_scheduler_error')}
        {renderPromethuesChart('tikv_coprocessor_error')}
        {renderPromethuesChart('tikv_grpc_message_error')}
        {renderPromethuesChart('tikv_leader_drop')}
        {renderPromethuesChart('tikv_leader_missing')}
      </CollpasePanel>
      <CollpasePanel title="Server" expand={false}>
        {renderPromethuesChart('tikv_channel_full')}
        {renderPromethuesChart('tikv_approximate_region_size')}
      </CollpasePanel>
      <CollpasePanel title="Raft IO" expand={false}>
        {renderPromethuesChart('tikv_apply_log_duration')}
        {renderPromethuesChart('tikv_apply_log_duration_per_server')}
        {renderPromethuesChart('tikv_append_log_duration')}
        {renderPromethuesChart('tikv_append_log_duration_per_server')}
      </CollpasePanel>
      <CollpasePanel title="Scheduler - prewrite" expand={false}>
        {renderPromethuesChart('tikv_scheduler_prewrite_latch_wait_duration')}
        {renderPromethuesChart('tivk_scheduler_prewrite_command_duration')}
      </CollpasePanel>
      <CollpasePanel title="Scheduler - commit" expand={false}>
        {renderPromethuesChart('tikv_scheduler_commit_latch_wait_duration')}
        {renderPromethuesChart('tivk_scheduler_commit_command_duration')}
      </CollpasePanel>
      <CollpasePanel title="Raft propose" expand={false}>
        {renderPromethuesChart('tikv_propose_wait_duration')}
      </CollpasePanel>
      <CollpasePanel title="Raft message" expand={false}>
        {renderPromethuesChart('tikv_raft_vote')}
      </CollpasePanel>
      <CollpasePanel title="Storage" expand={false}>
        {renderPromethuesChart('tikv_storage_async_write_duration')}
        {renderPromethuesChart('tikv_storage_async_snapshot_duration')}
      </CollpasePanel>
    </div>
  );
}

export default InspectionReport;
