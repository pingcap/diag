import React, { useState } from 'react';
import { Button } from 'antd';
import { IPromQuery } from './prom-query';

const styles = require('./prom-style.less');

interface PromChartHeaderProps {
  title: string;
  promQueries: IPromQuery[];
}

function PromChartHeader({ title, promQueries }: PromChartHeaderProps) {
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

export default PromChartHeader;
