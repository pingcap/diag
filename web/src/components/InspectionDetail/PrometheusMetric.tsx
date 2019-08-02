import React from 'react';
import { Collapse } from 'antd';

const styles = require('./AutoTable.less');

interface PrometheusMetricProps {
  title: string;
}

function PrometheusMetric({ title }: PrometheusMetricProps) {
  return (
    <div className={styles.table_container}>
      <Collapse defaultActiveKey="panel">
        <Collapse.Panel header={title} key="panel">
          TODO
        </Collapse.Panel>
      </Collapse>
    </div>
  );
}

export default PrometheusMetric;
