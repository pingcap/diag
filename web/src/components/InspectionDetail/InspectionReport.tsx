import React from 'react';
import { IInspectionDetail } from '@/models/inspection';
import AutoTable from './AutoTable';
import AutoObjectTable from './AutoObjectTable';
import PrometheusMetric from './PrometheusMetric';
import { fillInspectionId, PROM_SQLS } from '@/services/prometheus';

interface InspectionReportProps {
  inspection: IInspectionDetail;
}

function InspectionReport({ inspection }: InspectionReportProps) {
  const { report } = inspection;
  const inspectionId = inspection.uuid;

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
      <PrometheusMetric
        title="Vcores"
        promSQLs={[fillInspectionId(PROM_SQLS.vcores, inspectionId)]}
      />
      <PrometheusMetric
        title="Memory"
        promSQLs={[fillInspectionId(PROM_SQLS.memory, inspectionId)]}
      />
      <PrometheusMetric
        title="CPU Usage"
        promSQLs={[fillInspectionId(PROM_SQLS.cpu_usage, inspectionId)]}
      />
      <PrometheusMetric title="Load" promSQLs={[fillInspectionId(PROM_SQLS.load, inspectionId)]} />
      <PrometheusMetric
        title="Memorey Available"
        promSQLs={[fillInspectionId(PROM_SQLS.memory_available, inspectionId)]}
      />
      <PrometheusMetric
        title="Network Traffic"
        promSQLs={[
          fillInspectionId(PROM_SQLS.network_traffic_receive, inspectionId),
          fillInspectionId(PROM_SQLS.network_traffic_transmit, inspectionId),
        ]}
      />

      {/*

      <h3>2、PD</h3>
      <PrometheusMetric title="Cluster" />
      <PrometheusMetric title="Balance" />
      <PrometheusMetric title="Hot Region" />
      <PrometheusMetric title="Operator" />

      <h3>3、TiDB</h3>
      <PrometheusMetric title="Cluster" />
      <PrometheusMetric title="Balance" />
      <PrometheusMetric title="Hot Region" />
      <PrometheusMetric title="Operator" />

      <h3>4、TiKV</h3>
      <PrometheusMetric title="Cluster" />
      <PrometheusMetric title="Balance" />
      <PrometheusMetric title="Hot Region" />
      <PrometheusMetric title="Operator" /> */}
    </div>
  );
}

export default InspectionReport;
