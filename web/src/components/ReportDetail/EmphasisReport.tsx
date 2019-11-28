import React, { useState, useEffect } from 'react';
import moment from 'moment';
import _ from 'lodash';
import { oriRequest } from '@/utils/request';
import { IEmphasisDetail } from '@/models/emphasis';
import { EMPHASIS_DETAILS } from './report-detail-config';
import { IPromConfigSection } from '../Prom/prom-panel-config';
import { IPromParams, genPromParams } from '../Prom/prom-query';
import PromSection from '../Prom/PromSection';
import ReportSection from './ReportSection';

interface EmphasisReportProps {
  emphasis: IEmphasisDetail;
}

function genItemApiUrl(emphasisId: string, itemType: string) {
  return `/emphasis/${emphasisId}${itemType}`;
}

// https://www.lodashjs.com/docs/latest#_templatestring-options
// 使用自定义的模板分隔符
// _.templateSettings.interpolate = /{{([\s\S]+?)}}/g;
// var compiled = _.template('hello {{ user }}!');
// compiled({ 'user': 'mustache' });
// // => 'hello mustache!'
_.templateSettings.interpolate = /{{([\s\S]+?)}}/g;

function EmphasisReport({ emphasis }: EmphasisReportProps) {
  const start = moment(emphasis.investgating_start).unix();
  const end = moment(emphasis.investgating_end).unix();
  const promParams: IPromParams = genPromParams(start, end);
  const [emphasisPromConfig, setEmphasisConfig] = useState<IPromConfigSection | null>(null);

  useEffect(() => {
    oriRequest('/prom-emphasis.json').then(data => setEmphasisConfig(data));
  }, []);

  const exprConverter = (expr: string) => _.template(expr)({ inspectionId: emphasis.uuid });

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
          exprConverter={exprConverter}
        />
      )}
    </div>
  );
}

export default EmphasisReport;
