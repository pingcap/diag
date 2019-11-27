import React, { useEffect, useState } from 'react';
import moment from 'moment';
import _ from 'lodash';
import { oriRequest } from '@/utils/request';
import { IInspectionDetail } from '@/models/inspection';
import { IPromParams } from '@/services/prom-query';
import { INSPECTION_DETAILS } from './report-detail-config';
import { IPromConfigSection } from './prom-panel-config';
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
  const [inspectionPromDetail, setInspectionPromDetail] = useState<IPromConfigSection | null>(null);

  useEffect(() => {
    oriRequest('/prom-inspection.json').then(data => setInspectionPromDetail(data));
  }, []);

  const exprConverter = (expr: string) => _.template(expr)({ inspectionId: inspection.uuid });

  return (
    <div style={{ marginTop: 20 }}>
      <ReportSection
        reportDetailConfig={INSPECTION_DETAILS}
        fullApiUrlGenerator={(val: string) => genItemApiUrl(inspection.uuid, val)}
      />
      {inspectionPromDetail && (
        <PromSection
          promConfigSection={inspectionPromDetail}
          promParams={promParams}
          exprConverter={exprConverter}
        />
      )}
    </div>
  );
}

export default InspectionReport;
