import React from 'react';
import _ from 'lodash';
import { IPromParams, IPromQuery } from '@/services/prom-query';
import { IPromConfigSection, IPromConfigSubPanel } from './prom-panel-config';
import CollpasePanel from './CollapsePanel';
import PromChart from './PromChart';
import PromTable from './PromTable';

interface PromSectionProps {
  promConfigSection: IPromConfigSection;
  promParams: IPromParams;
  exprConverter?: (expr: string) => string;
}

function PromSection({ promConfigSection, promParams, exprConverter }: PromSectionProps) {
  function renderPromChart(subPanel: IPromConfigSubPanel) {
    const promQueries: IPromQuery[] = subPanel.targets.map(target => ({
      promQL: exprConverter ? exprConverter(target.expr) : target.expr,
      labelTemplate: target.legendFormat,
    }));

    if (subPanel.subPanelType === 'table') {
      return (
        <PromTable
          key={subPanel.subPanelKey}
          title={subPanel.title}
          tableColumns={subPanel.tableColumns || ['', '']}
          promQueries={promQueries}
          promParams={promParams}
          valUnit={subPanel.yaxis}
        />
      );
    }
    return (
      <PromChart
        key={subPanel.subPanelKey}
        title={subPanel.title}
        promQueries={promQueries}
        promParams={promParams}
        yaxis={subPanel.yaxis}
      />
    );
  }

  return (
    <div>
      <h2>{promConfigSection.title}</h2>
      {promConfigSection.panels.map(panel => (
        <CollpasePanel title={panel.title} key={panel.panelKey} expand={panel.expand || false}>
          {panel.subPanels.map(renderPromChart)}
        </CollpasePanel>
      ))}
    </div>
  );
}

export default PromSection;
