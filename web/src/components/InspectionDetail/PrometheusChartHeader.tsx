import React, { useState } from 'react';
import { Button } from 'antd';
import { IPromQuery } from '@/services/prometheus-config-charts';

const styles = require('./inspection-detail-style.less');

interface PrometheusChartHeaderProps {
  title: string;
  promQueries: IPromQuery[];
}

function PrometheusChartHeader({ title, promQueries }: PrometheusChartHeaderProps) {
  const [showPromQLs, setShowPromQLs] = useState(false);
  function renderPromSQLs() {
    return (
      <ul className={styles.prom_qls_list}>
        {promQueries.map(query => (
          <li key={query.promQL}>{query.promQL}</li>
        ))}
      </ul>
    );
  }
  return (
    <React.Fragment>
      <h4 style={{ textAlign: 'center', marginTop: 10 }}>
        {title}
        <Button
          icon={showPromQLs ? 'up-circle' : 'down-circle'}
          onClick={() => setShowPromQLs(!showPromQLs)}
          type="link"
        />
      </h4>
      {showPromQLs && renderPromSQLs()}
    </React.Fragment>
  );
}

export default PrometheusChartHeader;
