import React, { useState, useEffect } from 'react';
import moment from 'moment';
import _ from 'lodash';
import { oriRequest } from '@/utils/request';
import { IPromParams } from '@/services/prom-query';
import { IEmphasisDetail } from '@/models/emphasis';
import { EMPHASIS_DETAILS } from './report-detail-config';
import { IPromConfigSection } from './prom-panel-config';
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
  const [emphasisPromConfig, setEmphasisConfig] = useState<IPromConfigSection | null>(null);

  useEffect(() => {
    oriRequest('/prom-emphasis.json').then(data => setEmphasisConfig(data));
  }, []);

  return (
    <div style={{ marginTop: 20 }}>
      <ReportSection
        reportDetailConfig={EMPHASIS_DETAILS}
        fullApiUrlGenerator={(val: string) => genItemApiUrl(emphasis.uuid, val)}
      />
      {emphasisPromConfig && (
        <PromSection
          promConfigSection={emphasisPromConfig}
          promParams={promParams}
          inspectionId={emphasis.uuid}
        />
      )}
    </div>
  );
}

export default EmphasisReport;
