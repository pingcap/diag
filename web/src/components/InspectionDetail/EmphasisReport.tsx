import React from 'react';
import moment from 'moment';
import _ from 'lodash';
import { IPromParams } from '@/services/prometheus-query';
import { IEmphasisDetail } from '@/models/emphasis';
import { ReportDetailConfig, EMPHASIS_DETAILS } from '@/services/report-detail-config';
import { EMPHASIS_PROM_DETAIL } from '@/services/promtheus-panel-config';
import PromSection from './PromSection';
import AutoPanelTable from './AutoPanelTable';

interface EmphasisReportProps {
  emphasis: IEmphasisDetail;
}

const CHART_SAMPLE_COUNT = 15;

function genItemApiUrl(emphasisId: string, itemType: string) {
  return `/emphasis/${emphasisId}${itemType}`;
}

// TODO: 提取重复代码
function EmphasisReport({ emphasis }: EmphasisReportProps) {
  const start = moment(emphasis.investgating_start).unix();
  const end = moment(emphasis.investgating_end).unix();
  const step = Math.floor((end - start) / CHART_SAMPLE_COUNT);
  const promParams: IPromParams = { start, end, step };

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

  return (
    <div style={{ marginTop: 20 }}>
      {renderNormalSections(EMPHASIS_DETAILS)}
      <PromSection
        promConfigSection={EMPHASIS_PROM_DETAIL}
        promParams={promParams}
        inspectionId={emphasis.uuid}
      />
    </div>
  );
}

export default EmphasisReport;
