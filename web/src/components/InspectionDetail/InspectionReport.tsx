import React from 'react';
import moment from 'moment';
import _ from 'lodash';
import { IInspectionDetail } from '@/models/inspection';
import { IPromParams } from '@/services/prometheus-query';
import { INSPECTION_DETAILS, ReportDetailConfig } from '@/services/report-detail-config';
import { INSPECTION_PROM_DETAIL } from '@/services/promtheus-panel-config';
import PromSection from './PromSection';
import AutoPanelTable from './AutoPanelTable';

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

  return (
    <div style={{ marginTop: 20 }}>
      {renderNormalSections(INSPECTION_DETAILS)}
      <PromSection
        promConfigSection={INSPECTION_PROM_DETAIL}
        promParams={promParams}
        inspectionId={inspection.uuid}
      />
    </div>
  );
}

export default InspectionReport;
