import React from 'react';
import moment from 'moment';
import _ from 'lodash';
// import AutoTable from './AutoTable';
import { IPromParams } from '@/services/prometheus-query';
import CollpasePanel from './CollapsePanel';
import PrometheusChart from './PrometheusChart';
import PrometheusTable from './PrometheusTable';
import { IPromQuery, PROM_CHARTS } from '@/services/prometheus-config-charts';
import {
  IPanel,
  ALL_PANELS,
  EMPHASIS_DB_PERFORMANCE_PANELS,
} from '@/services/prometheus-config-panels';
import { IEmphasisDetail } from '@/models/emphasis';

interface EmphasisReportProps {
  emphasis: IEmphasisDetail | undefined;
}

const CHART_SAMPLE_COUNT = 15;

// TODO: 提取重复代码
function EmphasisReport({ emphasis }: EmphasisReportProps) {
  // const start = moment(emphasis.scrape_begin).unix();
  const start = moment().unix();
  // const end = moment(emphasis.scrape_end).unix();
  const end = start + 5 * 60;
  const step = Math.floor((end - start) / CHART_SAMPLE_COUNT);
  const promParams: IPromParams = { start, end, step };

  function renderPromethuesChart(chartKey: string) {
    const promChart = PROM_CHARTS[chartKey];
    const promQueries: IPromQuery[] = promChart.queries.map(promQuery => ({
      ...promQuery,
      // promQL: _.template(promQuery.promQLTemplate)({ inspectionId: emphasis.uuid }),
      promQL: _.template(promQuery.promQLTemplate)({ inspectionId: 'aaa' }),
    }));
    if (promChart.chartType === 'table') {
      return (
        <PrometheusTable
          key={chartKey}
          title={promChart.title}
          tableColumns={promChart.tableColumns || ['', '']}
          promQueries={promQueries}
          promParams={promParams}
        />
      );
    }
    return (
      <PrometheusChart
        key={chartKey}
        title={promChart.title}
        promQueries={promQueries}
        promParams={promParams}
      />
    );
  }

  function renderPanel(panelKey: string) {
    const panel: IPanel = ALL_PANELS[panelKey];
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
      <h2>一、问题定位</h2>
      {/* <AutoTable title="overview" apiUrl={`/emphasis/${emphasis.uuid}/symptom`} /> */}

      <h2>二、问题排查监控信息</h2>
      {renderPanels(EMPHASIS_DB_PERFORMANCE_PANELS)}
    </div>
  );
}

export default EmphasisReport;
