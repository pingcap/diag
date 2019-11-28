import React, { useEffect, useState } from 'react';
import moment from 'moment';
import _ from 'lodash';
import { oriRequest } from '@/utils/request';
import { IInspectionDetail } from '@/models/inspection';
import { INSPECTION_DETAILS } from './report-detail-config';
import { IPromConfigSection } from '../Prom/prom-panel-config';
import { IPromParams, genPromParams } from '../Prom/prom-query';
import PromSection from '../Prom/PromSection';
import ReportSection from './ReportSection';

interface InspectionReportProps {
  inspection: IInspectionDetail;
}

function genItemApiUrl(inspectionId: string, itemType: string) {
  return `/inspections/${inspectionId}${itemType}`;
}

// https://www.lodashjs.com/docs/latest#_templatestring-options
// 使用自定义的模板分隔符
// _.templateSettings.interpolate = /{{([\s\S]+?)}}/g;
// var compiled = _.template('hello {{ user }}!');
// compiled({ 'user': 'mustache' });
// // => 'hello mustache!'
_.templateSettings.interpolate = /{{([\s\S]+?)}}/g;

function InspectionReport({ inspection }: InspectionReportProps) {
  const start = moment(inspection.scrape_begin).unix();
  const end = moment(inspection.scrape_end).unix();
  const promParams: IPromParams = genPromParams(start, end);
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
