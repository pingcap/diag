import React from 'react';
import moment from 'moment';
import { IInspectionDetail } from '@/models/inspection';
import AutoTable from './AutoTable';
import AutoObjectTable from './AutoObjectTable';
import { fillInspectionId, PROM_SQLS, IPromParams } from '@/services/prometheus';
import CollpasePanel from './CollapsePanel';
import PrometheusChart from './PrometheusChart';

interface InspectionReportProps {
  inspection: IInspectionDetail;
}

const CHART_SAMPLE_COUNT = 15;

function InspectionReport({ inspection }: InspectionReportProps) {
  const { report } = inspection;

  const fillHelper = (oriPromSQL: string | string[]) =>
    fillInspectionId(oriPromSQL, inspection.uuid);

  const start = moment(inspection.scrape_begin).unix();
  const end = moment(inspection.scrape_end).unix();
  const step = Math.floor((end - start) / CHART_SAMPLE_COUNT);
  const promParams: IPromParams = { start, end, step };

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
      <CollpasePanel title="Vcores">
        <PrometheusChart promSQLs={fillHelper(PROM_SQLS.vcores)} promParams={promParams} />
      </CollpasePanel>
      <CollpasePanel title="Memory">
        <PrometheusChart promSQLs={fillHelper(PROM_SQLS.memory)} promParams={promParams} />
      </CollpasePanel>
      <CollpasePanel title="CPU Usage">
        <PrometheusChart promSQLs={fillHelper(PROM_SQLS.cpu_usage)} promParams={promParams} />
      </CollpasePanel>
      <CollpasePanel title="Load">
        <PrometheusChart promSQLs={fillHelper(PROM_SQLS.load)} promParams={promParams} />
      </CollpasePanel>
      <CollpasePanel title="Memorey Available">
        <PrometheusChart
          promSQLs={fillHelper(PROM_SQLS.memory_available)}
          promParams={promParams}
        />
      </CollpasePanel>
      <CollpasePanel title="Network Traffic">
        <PrometheusChart promSQLs={fillHelper(PROM_SQLS.network_traffic)} promParams={promParams} />
      </CollpasePanel>
      <CollpasePanel title="TCP Retrans">
        <PrometheusChart promSQLs={fillHelper(PROM_SQLS.tcp_retrans)} promParams={promParams} />
      </CollpasePanel>
      <CollpasePanel title="IO Util">
        <PrometheusChart promSQLs={fillHelper(PROM_SQLS.io_util)} promParams={promParams} />
      </CollpasePanel>

      <h3>2、PD</h3>
      <CollpasePanel title="Cluster">
        <PrometheusChart
          title="Store Status"
          promSQLs={fillHelper(PROM_SQLS.stores_status)}
          promParams={promParams}
        />
      </CollpasePanel>

      <h3>3、TiDB</h3>

      <h3>4、TiKV</h3>
    </div>
  );
}

export default InspectionReport;
