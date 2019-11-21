import React from 'react';
import moment from 'moment';
import _ from 'lodash';
import { IPromParams } from '@/services/prom-query';
import { IEmphasisDetail } from '@/models/emphasis';
import { EMPHASIS_DETAILS } from '@/services/report-detail-config';
import { EMPHASIS_PROM_DETAIL } from '@/services/prom-panel-config';
import PromSection from './PromSection';
import ReportSection from './ReportSection';

interface EmphasisReportProps {
  emphasis: IEmphasisDetail;
}

const CHART_SAMPLE_COUNT = 15;

function genItemApiUrl(emphasisId: string, itemType: string) {
  return `/emphasis/${emphasisId}${itemType}`;
}

function EmphasisReport({ emphasis }: EmphasisReportProps) {
  const start = moment(emphasis.investgating_start).unix();
  const end = moment(emphasis.investgating_end).unix();
  const step = Math.floor((end - start) / CHART_SAMPLE_COUNT);
  const promParams: IPromParams = { start, end, step };

  return (
    <div style={{ marginTop: 20 }}>
      <ReportSection
        reportDetailConfig={EMPHASIS_DETAILS}
        fullApiUrlGenerator={(val: string) => genItemApiUrl(emphasis.uuid, val)}
      />
      <PromSection
        promConfigSection={EMPHASIS_PROM_DETAIL}
        promParams={promParams}
        inspectionId={emphasis.uuid}
      />
    </div>
  );
}

export default EmphasisReport;
