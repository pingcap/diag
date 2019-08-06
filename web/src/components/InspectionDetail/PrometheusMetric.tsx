import React from 'react';
import { Collapse } from 'antd';
import PrometheusChart from './PrometheusChart';

const styles = require('./AutoTable.less');

interface PrometheusMetricProps {
  title: string;
  promSQLs: string[];
}

function PrometheusMetric({ title, promSQLs }: PrometheusMetricProps) {
  return (
    <div className={styles.table_container}>
      <Collapse defaultActiveKey="panel">
        <Collapse.Panel header={<strong>{title}</strong>} key="panel">
          <PrometheusChart promSQLs={promSQLs} />
        </Collapse.Panel>
      </Collapse>
    </div>
  );
}

export default PrometheusMetric;
