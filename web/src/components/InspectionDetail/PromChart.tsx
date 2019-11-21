import React from 'react';
import SerialLineChart from '../Chart/SerialLineChart';
import { IPromParams } from '@/services/prometheus-query';
import { IPromQuery } from '@/services/prometheus-config-charts';
import { usePromQueries } from './use-prom-queries';
import PromChartHeader from './PromChartHeader';
import { IPromConfigYaxis } from '@/services/promtheus-panel-config';

const styles = require('./inspection-detail-style.less');

interface PromChartProps {
  title: string;

  promQueries: IPromQuery[];
  promParams: IPromParams;
  yaxis: IPromConfigYaxis;
}

function PromChart({ title, promQueries, promParams, yaxis }: PromChartProps) {
  const [loading, chartLabels, oriChartData] = usePromQueries(promQueries, promParams);

  return (
    <div className={styles.chart_container}>
      <PromChartHeader title={title} promQueries={promQueries} />

      {loading && <p style={{ textAlign: 'center' }}>loading...</p>}
      {!loading && oriChartData.length === 0 && <p style={{ textAlign: 'center' }}>No Data</p>}
      {!loading && oriChartData.length > 0 && (
        <div style={{ height: 200 }}>
          <SerialLineChart data={oriChartData} labels={chartLabels} yaxis={yaxis} />
        </div>
      )}
    </div>
  );
}

export default PromChart;
