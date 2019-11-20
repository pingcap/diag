import React from 'react';
import moment from 'moment';
import _ from 'lodash';
import { IInspectionDetail } from '@/models/inspection';
import AutoPanelTable from './AutoPanelTable';
import { IPromParams } from '@/services/prometheus-query';
import CollpasePanel from './CollapsePanel';
import PrometheusChart from './PrometheusChart';
import PrometheusTable from './PrometheusTable';
import { IPromQuery } from '@/services/prometheus-config-charts';
import { INSPECTION_DETAILS, ReportDetailConfig } from '@/services/report-detail-config';
import {
  IPromConfigSection,
  INSPECTION_PROM_DETAIL,
  IPromConfigSubPanel,
} from '@/services/promtheus-panel-config';
import { getValueFormat } from 'value-formats';

interface InspectionReportProps {
  inspection: IInspectionDetail;
}

const CHART_SAMPLE_COUNT = 15;

function genItemApiUrl(inspectionId: string, itemType: string) {
  return `/inspections/${inspectionId}${itemType}`;
}

_.templateSettings.interpolate = /{{([\s\S]+?)}}/g;

function InspectionReport({ inspection }: InspectionReportProps) {
  const start = moment(inspection.scrape_begin).unix();
  const end = moment(inspection.scrape_end).unix();
  const step = Math.floor((end - start) / CHART_SAMPLE_COUNT);
  const promParams: IPromParams = { start, end, step };

  function renderPromChart(subPanel: IPromConfigSubPanel) {
    const promQueries: IPromQuery[] = subPanel.targets.map(target => ({
      promQL: _.template(target.expr)({ inspectionId: inspection.uuid }),
      labelTemplate: target.legendFormat,
    }));

    // TODO: extract
    const formatFunc = getValueFormat(subPanel.yaxis.format);
    const valConverter = (val: number) => formatFunc(val, subPanel.yaxis.decimals || 2);

    // if (promChart.chartType === 'table') {
    //   return (
    //     <PrometheusTable
    //       key={chartKey}
    //       title={promChart.title}
    //       tableColumns={promChart.tableColumns || ['', '']}
    //       promQueries={promQueries}
    //       promParams={promParams}
    //     />
    //   );
    // }
    return (
      <PrometheusChart
        key={subPanel.subPanelKey}
        title={subPanel.title}
        promQueries={promQueries}
        promParams={promParams}
        valConverter={valConverter}
      />
    );
  }

  // TODO: extract to individual component
  function renderNormalSections(config: ReportDetailConfig) {
    return config.map(section => (
      <div key={section.sectionKey}>
        <h2>{section.title}</h2>
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

  function renderPromSections(config: IPromConfigSection) {
    return (
      <div>
        <h2>{config.title}</h2>
        {config.panels.map(panel => (
          <CollpasePanel title={panel.title} key={panel.panelKey} expand={panel.expand || false}>
            {panel.subPanels.map(renderPromChart)}
          </CollpasePanel>
        ))}
      </div>
    );
  }

  return (
    <div style={{ marginTop: 20 }}>
      {renderNormalSections(INSPECTION_DETAILS)}
      {renderPromSections(INSPECTION_PROM_DETAIL)}
    </div>
  );
}

export default InspectionReport;
