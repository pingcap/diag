import React, { useState } from 'react';
import { Table, Collapse } from 'antd';
import _ from 'lodash';
import { useReportItemQuery } from './use-report-item-query';

const styles = require('./AutoTable.less');

interface AutoTableProps {
  title?: string;
  expand?: boolean;

  apiUrl: string;
}

function AutoTable({ title, expand = true, apiUrl }: AutoTableProps) {
  const [collapsed, setCollapsed] = useState(!expand);

  const [conclusion, tableColumns, dataSource] = useReportItemQuery(apiUrl);

  function handleCollapseChange(expandKeys: string[] | string) {
    setCollapsed(expandKeys.length === 0);
  }

  function renderHeader() {
    return (
      <strong>
        {title} {collapsed && `(${dataSource.length})`}
      </strong>
    );
  }

  function renderConclusion() {
    return (
      <ul className={styles.conclusion_list}>
        {conclusion.map(item => (
          <li key={`${item.message}`}>
            {item.status === 'abnormal' ? (
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
                  scroll={_.endsWith(apiUrl, '/dmesg') ? { x: true, y: 600 } : { x: true }}
                />
              </React.Fragment>
            ))}
        </Collapse.Panel>
      </Collapse>
    </div>
  );
}

export default AutoTable;
