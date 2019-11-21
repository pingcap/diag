import React from 'react';
import _ from 'lodash';
import { ReportDetailConfig } from '@/services/report-detail-config';
import AutoPanelTable from './AutoPanelTable';

interface ReportSectionProps {
  reportDetailConfig: ReportDetailConfig;
  fullApiUrlGenerator: (partialApiUrl: string) => string;
}

function ReportSection({ reportDetailConfig, fullApiUrlGenerator }: ReportSectionProps) {
  return (
    <div>
      {reportDetailConfig.map(section => (
        <div key={section.sectionKey}>
          <h2>{section.title}</h2>
          {section.panels.map(panel => (
            <AutoPanelTable
              key={panel.apiUrl}
              fullApiUrl={fullApiUrlGenerator(panel.apiUrl)}
              panelConfig={panel}
            />
          ))}
        </div>
      ))}
    </div>
  );
}

export default ReportSection;
