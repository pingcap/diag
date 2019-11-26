import React, { useState } from 'react';
import { Table, Collapse } from 'antd';
import _ from 'lodash';
import { useReportItemQuery } from './use-report-item-query';
import { IReportDetailConfigPanel } from './report-detail-config';

const styles = require('./report-detail-style.less');

interface AutoPanelTableProps {
  fullApiUrl: string;

  panelConfig: IReportDetailConfigPanel;
}

function AutoPanelTable({ fullApiUrl, panelConfig }: AutoPanelTableProps) {
  const [collapsed, setCollapsed] = useState(panelConfig.collapse || false);

  const [conclusion, tableColumns, dataSource, hasAbnormal] = useReportItemQuery(
    fullApiUrl,
    panelConfig.dataType,
    panelConfig.lessMoreColumns || [],
    panelConfig.columnsUnit || {},
  );

  function handleCollapseChange(expandKeys: string[] | string) {
    setCollapsed(expandKeys.length === 0);
  }

  function renderHeader() {
    return (
      <strong style={{ color: panelConfig.titleColor || (hasAbnormal ? 'red' : 'inherit') }}>
        {panelConfig.title} {collapsed && `(${dataSource.length})`}
      </strong>
    );
  }

  function renderConclusion() {
    return (
      <ul className={styles.conclusion_list}>
        {conclusion.map(item => (
          <li key={`${item.message}`}>
            {item.status === 'error' ? (
              <p style={{ color: 'red', fontSize: '16px' }}>{item.message}</p>
            ) : (
              <p>{item.message}</p>
            )}
          </li>
        ))}
      </ul>
    );
  }

  return (
    <div className={styles.table_container}>
      <Collapse defaultActiveKey={collapsed ? '' : 'panel'} onChange={handleCollapseChange}>
        <Collapse.Panel header={renderHeader()} key="panel">
          {!collapsed &&
            (tableColumns.length === 0 ? (
              'None'
            ) : (
              <React.Fragment>
                {conclusion.length > 0 && renderConclusion()}
                <Table
                  dataSource={dataSource}
                  columns={tableColumns}
                  pagination={false}
                  scroll={panelConfig.limitHeight ? { y: panelConfig.height || 400 } : {}}
                />
              </React.Fragment>
            ))}
        </Collapse.Panel>
      </Collapse>
    </div>
  );
}

export default AutoPanelTable;
