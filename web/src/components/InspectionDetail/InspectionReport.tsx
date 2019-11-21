import React from 'react';
import moment from 'moment';
import _ from 'lodash';
import { IInspectionDetail } from '@/models/inspection';
import { IPromParams } from '@/services/prometheus-query';
import { INSPECTION_DETAILS } from '@/services/report-detail-config';
import { INSPECTION_PROM_DETAIL } from '@/services/promtheus-panel-config';
import PromSection from './PromSection';
import ReportSection from './ReportSection';

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

  return (
    <div style={{ marginTop: 20 }}>
      <ReportSection
        reportDetailConfig={INSPECTION_DETAILS}
        fullApiUrlGenerator={(val: string) => genItemApiUrl(inspection.uuid, val)}
      />
      <PromSection
        promConfigSection={INSPECTION_PROM_DETAIL}
        promParams={promParams}
        inspectionId={inspection.uuid}
      />
    </div>
  );
}

export default InspectionReport;
