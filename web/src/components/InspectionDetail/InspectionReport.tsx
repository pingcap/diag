import React from 'react';
import moment from 'moment';
import _ from 'lodash';
import { IInspectionDetail } from '@/models/inspection';
import AutoTable from './AutoTable';
import { IPromParams } from '@/services/prometheus-query';
import CollpasePanel from './CollapsePanel';
import PrometheusChart from './PrometheusChart';
import PrometheusTable from './PrometheusTable';
import {
  IPromQuery,
  PROM_CHARTS,
  PANELS,
  IPanel,
  TIKV_PANELS,
  TIDB_PANELS,
  PD_PANELS,
  GLOBAL_PANNEL,
} from '@/services/prometheus-config';

interface InspectionReportProps {
  inspection: IInspectionDetail;
}

const CHART_SAMPLE_COUNT = 15;

function genItemApiUrl(inspectionId: string, itemType: string) {
  return `/inspections/${inspectionId}/${itemType}`;
}

function InspectionReport({ inspection }: InspectionReportProps) {
  const start = moment(inspection.scrape_begin).unix();
  const end = moment(inspection.scrape_end).unix();
  const step = Math.floor((end - start) / CHART_SAMPLE_COUNT);
  const promParams: IPromParams = { start, end, step };

  function renderPromethuesChart(chartKey: string) {
    const promChart = PROM_CHARTS[chartKey];
    const title = promChart.showTitle === false ? undefined : promChart.title;
    const promQueries: IPromQuery[] = promChart.queries.map(promQuery => ({
      ...promQuery,
      promQL: _.template(promQuery.promQLTemplate)({ inspectionId: inspection.uuid }),
    }));
    if (promChart.chartType === 'table') {
      return (
        <PrometheusTable
          key={chartKey}
          title={title}
          tableColumns={promChart.tableColumns || ['', '']}
          promQueries={promQueries}
          promParams={promParams}
        />
      );
    }
    return (
      <PrometheusChart
        key={chartKey}
        title={title}
        promQueries={promQueries}
        promParams={promParams}
      />
    );
  }

  function renderPanel(panelKey: string) {
    const panel: IPanel = PANELS[panelKey];
    return (
      <CollpasePanel title={panel.title} expand={panel.expand || false} key={panelKey}>
        {panel.charts.map(renderPromethuesChart)}
      </CollpasePanel>
    );
  }

  function renderPanels(panelKeys: string[]) {
    return panelKeys.map(renderPanel);
  }

  return (
    <div style={{ marginTop: 20 }}>
      <h2>一、全局诊断</h2>
      <AutoTable title="overview" apiUrl={`/inspections/${inspection.uuid}/symptom`} />

      <h2>二、基本信息</h2>
      <AutoTable title="1、基本信息" apiUrl={`/inspections/${inspection.uuid}/basic`} />
      <AutoTable
        title="2、数据库基本信息"
        apiUrl={genItemApiUrl(inspection.uuid, 'dbinfo')}
        expand={false}
      />
      <AutoTable
        title="3、资源信息 (使用率%)"
        apiUrl={genItemApiUrl(inspection.uuid, 'resource')}
      />
      <AutoTable title="4、告警信息" apiUrl={genItemApiUrl(inspection.uuid, 'alert')} />
      <AutoTable title="5、慢查询信息" apiUrl={genItemApiUrl(inspection.uuid, 'slowlog')} />
      <AutoTable title="6、硬件信息" apiUrl={genItemApiUrl(inspection.uuid, 'hardware')} />
      <AutoTable title="7、软件信息" apiUrl={genItemApiUrl(inspection.uuid, 'software')} />
      <AutoTable
        title="8、软件配置信息"
        apiUrl={genItemApiUrl(inspection.uuid, 'config')}
        expand={false}
      />
      <AutoTable title="9、机器 NTP 时间同步信息" apiUrl={genItemApiUrl(inspection.uuid, 'ntp')} />
      <AutoTable title="10、网络质量信息" apiUrl={genItemApiUrl(inspection.uuid, 'network')} />
      <AutoTable title="11、集群拓扑结构信息" apiUrl={genItemApiUrl(inspection.uuid, 'topology')} />
      <AutoTable
        title="12、dmesg 信息"
        apiUrl={genItemApiUrl(inspection.uuid, 'dmesg')}
        expand={false}
      />

      <h2>三、监控信息</h2>
      <h3>1、全局监控</h3>
      {renderPanels(GLOBAL_PANNEL)}
      <h3>2、PD</h3>
      {renderPanels(PD_PANELS)}
      <h3>3、TiDB</h3>
      {renderPanels(TIDB_PANELS)}
      <h3>4、TiKV</h3>
      {renderPanels(TIKV_PANELS)}
    </div>
  );
}

export default InspectionReport;
