import React from 'react';
import _ from 'lodash';
import { IPromParams } from '@/services/prometheus-query';
import { IPromQuery } from '@/services/prometheus-config-charts';
import { IPromConfigSection, IPromConfigSubPanel } from '@/services/promtheus-panel-config';
import CollpasePanel from './CollapsePanel';
import PromChart from './PromChart';
import PromTable from './PromTable';

interface PromSectionProps {
  promConfigSection: IPromConfigSection;
  promParams: IPromParams;
  inspectionId: string;
}

// https://www.lodashjs.com/docs/latest#_templatestring-options
// 使用自定义的模板分隔符
// _.templateSettings.interpolate = /{{([\s\S]+?)}}/g;
// var compiled = _.template('hello {{ user }}!');
// compiled({ 'user': 'mustache' });
// // => 'hello mustache!'
_.templateSettings.interpolate = /{{([\s\S]+?)}}/g;

function PromSection({ promConfigSection, promParams, inspectionId }: PromSectionProps) {
  function renderPromChart(subPanel: IPromConfigSubPanel) {
    const promQueries: IPromQuery[] = subPanel.targets.map(target => ({
      promQL: _.template(target.expr)({ inspectionId }),
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
