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
    return <PrometheusChart title={title} promMetrics={metrics} promParams={promParams} />;
  }

  return (
    <div style={{ marginTop: 20 }}>
      <h2>一、全局诊断</h2>
      <AutoTable title="overview" dataArr={report.symptoms} />

      <h2>二、基本信息</h2>
      <AutoObjectTable title="1、基本信息" data={report.basic || {}} />
      <AutoTable title="2、数据库基本信息" dataArr={report.dbinfo || []} />
      <AutoTable title="3、资源信息" dataArr={report.resource || []} />
      <AutoTable title="4、告警信息" dataArr={report.alert || []} />
      <AutoTable title="5、慢查询信息" dataArr={report.slow_log || []} />
      <AutoTable title="6、硬件信息" dataArr={report.hardware || []} />
      <AutoTable title="7、软件信息" dataArr={report.software_version || []} />
      <AutoTable title="8、软件配置信息" dataArr={report.software_config || []} />
      <AutoTable title="9、机器 NTP 时间同步信息" dataArr={[]} />
      <AutoTable title="10、网络质量信息" dataArr={report.network || []} />
      <AutoTable title="11、集群拓扑结构信息" dataArr={[]} />
      <AutoTable title="12、demsg 信息" dataArr={report.demsg || []} />

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

      <h3>4、TiKV</h3>
    </div>
  );
}

export default InspectionReport;
