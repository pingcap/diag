import React from 'react';
import moment from 'moment';
import _ from 'lodash';
import AutoPanelTable from './AutoPanelTable';
import { IPromParams } from '@/services/prometheus-query';
import CollpasePanel from './CollapsePanel';
import PrometheusChart from './PrometheusChart';
import PrometheusTable from './PrometheusTable';
import { IPromQuery } from '@/services/prometheus-config-charts';
import { IEmphasisDetail } from '@/models/emphasis';
import { ReportDetailConfig, EMPHASIS_DETAILS } from '@/services/report-detail-config';
import {
  IPromConfigSection,
  EMPHASIS_PROM_DETAIL,
  IPromConfigSubPanel,
} from '@/services/promtheus-panel-config';
import { getValueFormat } from 'value-formats';

interface EmphasisReportProps {
  emphasis: IEmphasisDetail;
}

const CHART_SAMPLE_COUNT = 15;

function genItemApiUrl(emphasisId: string, itemType: string) {
  return `/emphasis/${emphasisId}${itemType}`;
}

_.templateSettings.interpolate = /{{([\s\S]+?)}}/g;

// TODO: 提取重复代码
function EmphasisReport({ emphasis }: EmphasisReportProps) {
  const start = moment(emphasis.investgating_start).unix();
  const end = moment(emphasis.investgating_end).unix();
  const step = Math.floor((end - start) / CHART_SAMPLE_COUNT);
  const promParams: IPromParams = { start, end, step };

  function renderPromChart(subPanel: IPromConfigSubPanel) {
    const promQueries: IPromQuery[] = subPanel.targets.map(target => ({
      promQL: _.template(target.expr)({ inspectionId: emphasis.uuid }),
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

  function renderNormalSections(config: ReportDetailConfig) {
    return config.map(section => (
      <div key={section.sectionKey}>
        <h2>{section.title}</h2>
        {section.panels.map(panel => (
          <AutoPanelTable
            key={panel.apiUrl}
            fullApiUrl={genItemApiUrl(emphasis.uuid, panel.apiUrl)}
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
      {renderNormalSections(EMPHASIS_DETAILS)}
      {renderPromSections(EMPHASIS_PROM_DETAIL)}
    </div>
  );
}

export default EmphasisReport;
