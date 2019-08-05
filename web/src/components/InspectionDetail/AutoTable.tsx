import React, { useMemo, useState } from 'react';
import { Table, Collapse } from 'antd';

const styles = require('./AutoTable.less');

interface AutoTableProps {
  title?: string;
  dataArr: object[];
}

function AutoTable({ title, dataArr }: AutoTableProps) {
  const [collapsed, setCollapsed] = useState(false);

  const tableColumns = useMemo(
    () =>
      Object.keys(dataArr[0] || {}).map(key => ({
        title: key,
        dataIndex: key,
        key,
      })),
    [dataArr],
  );
  const dataSource = useMemo(() => dataArr.map((item, index) => ({ ...item, key: `${index}` })), [
    dataArr,
  ]);

  function handleCollapseChange(expandKeys: string[] | string) {
    setCollapsed(expandKeys.length === 0);
  }

  function renderHeader() {
    return (
      <strong>
        {title} {collapsed && `(${dataArr.length})`}
      </strong>
    );
  }

  return (
    <div className={styles.table_container}>
      <Collapse defaultActiveKey="panel" onChange={handleCollapseChange}>
        <Collapse.Panel header={renderHeader()} key="panel">
          {dataArr.length === 0 ? (
            'None'
          ) : (
            <Table dataSource={dataSource} columns={tableColumns} pagination={false} />
          )}
        </Collapse.Panel>
      </Collapse>
    </div>
  );
}

export default AutoTable;
