import React from 'react';
import { Collapse } from 'antd';
import PrometheusChart from './PrometheusChart';

const styles = require('./AutoTable.less');

interface PrometheusMetricProps {
  title: string;
  promSQLStr: string;
}

function PrometheusMetric({ title, promSQLStr }: PrometheusMetricProps) {
  return (
    <div className={styles.table_container}>
      <Collapse defaultActiveKey="panel">
        <Collapse.Panel header={<strong>{title}</strong>} key="panel">
          <PrometheusChart promSQLStr={promSQLStr} />
        </Collapse.Panel>
      </Collapse>
    </div>
  );
}

export default PrometheusMetric;
