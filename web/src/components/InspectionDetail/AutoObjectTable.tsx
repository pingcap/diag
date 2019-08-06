import React, { useMemo, useState } from 'react';
import { Table, Collapse } from 'antd';

const styles = require('./AutoTable.less');

interface AutoTableProps {
  title?: string;
  data: object;
}

const tableColumns = [
  { title: '信息', dataIndex: 'field', key: 'field', render: (text: any) => <pre>{text}</pre> },
  { title: '值', dataIndex: 'value', key: 'value', render: (text: any) => <pre>{text}</pre> },
];

function AutoObjectTable({ title, data }: AutoTableProps) {
  const [collapsed, setCollapsed] = useState(false);

  const dataSource = useMemo(
    () =>
      Object.keys(data).map(key => ({
        field: key,
        value: data[key],
        key,
      })),
    [data],
  );

  function handleCollapseChange(expandKeys: string[] | string) {
    setCollapsed(expandKeys.length === 0);
  }

  function renderHeader() {
    return (
      <strong>
        {title} {collapsed && `(${Object.keys(data).length})`}
      </strong>
    );
  }

  return (
    <div className={styles.table_container}>
      <Collapse defaultActiveKey="panel" onChange={handleCollapseChange}>
        <Collapse.Panel header={renderHeader()} key="panel">
          <Table dataSource={dataSource} columns={tableColumns} pagination={false} />
        </Collapse.Panel>
      </Collapse>
    </div>
  );
}

export default AutoObjectTable;
