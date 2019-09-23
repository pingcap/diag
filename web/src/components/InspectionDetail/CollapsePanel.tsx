import React, { ReactNode, useState } from 'react';
import { Collapse } from 'antd';

const styles = require('./inspection-detail-style.less');

interface CollpasePanelProps {
  expand?: boolean;
  title: string;
  children: ReactNode;
}

function CollpasePanel({ expand = true, title, children }: CollpasePanelProps) {
  const [collapsed, setCollapsed] = useState(!expand);

  function handleCollapseChange(expandKeys: string[] | string) {
    setCollapsed(expandKeys.length === 0);
  }

  return (
    <div className={styles.table_container}>
      <Collapse defaultActiveKey={expand ? 'panel' : ''} onChange={handleCollapseChange}>
        <Collapse.Panel header={<strong>{title}</strong>} key="panel">
          {!collapsed && children}
        </Collapse.Panel>
      </Collapse>
    </div>
  );
}

export default CollpasePanel;
