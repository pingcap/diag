import React from 'react';
import moment from 'moment';
import _ from 'lodash';
import { IInspectionDetail } from '@/models/inspection';
import AutoPanelTable from './AutoPanelTable';
import { IPromParams } from '@/services/prometheus-query';
import CollpasePanel from './CollapsePanel';
import PrometheusChart from './PrometheusChart';
import PrometheusTable from './PrometheusTable';
import { IPromQuery, PROM_CHARTS } from '@/services/prometheus-config-charts';
import { INSPECTION_DETAILS } from '@/services/report-detail-config';
import {
  IPanel,
  ALL_PANELS,
  // TIKV_PANELS,
  // TIDB_PANELS,
  // PD_PANELS,
  // GLOBAL_PANNELS,
  DBA_PANELS,
} from '@/services/prometheus-config-panels';

interface InspectionReportProps {
  inspection: IInspectionDetail;
}

const CHART_SAMPLE_COUNT = 15;

function genItemApiUrl(inspectionId: string, itemType: string) {
  return `/inspections/${inspectionId}${itemType}`;
}

function InspectionReport({ inspection }: InspectionReportProps) {
  const start = moment(inspection.scrape_begin).unix();
  const end = moment(inspection.scrape_end).unix();
  const step = Math.floor((end - start) / CHART_SAMPLE_COUNT);
  const promParams: IPromParams = { start, end, step };

  function renderPromethuesChart(chartKey: string) {
    const promChart = PROM_CHARTS[chartKey];
    const promQueries: IPromQuery[] = promChart.queries.map(promQuery => ({
      ...promQuery,
      promQL: _.template(promQuery.promQLTemplate)({ inspectionId: inspection.uuid }),
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

  function renderPromPanel(panelKey: string) {
    const panel: IPanel = ALL_PANELS[panelKey];
    return (
      <CollpasePanel title={panel.title} expand={panel.expand || false} key={panelKey}>
        {panel.charts.map(renderPromethuesChart)}
      </CollpasePanel>
    );
  }

  function renderPromPanels(panelKeys: string[]) {
    return panelKeys.map(renderPromPanel);
  }

  function renderNormalSections() {
    return INSPECTION_DETAILS.map(section => (
      <div key={section.sectionKey}>
        <h2>{section.sectionTitle}</h2>
        {section.panels.map(panel => (
          <AutoPanelTable
            key={panel.apiUrl}
            fullApiUrl={genItemApiUrl(inspection.uuid, panel.apiUrl)}
            panelConfig={panel}
          />
        ))}
      </div>
    ));
  }

  return (
    <div style={{ marginTop: 20 }}>
      {renderNormalSections()}

      <h2>三、监控信息</h2>
      {renderPromPanels(DBA_PANELS)}
    </div>
  );
}

export default InspectionReport;
