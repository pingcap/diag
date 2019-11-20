import React from 'react';
import SerialLineChart from '../Chart/SerialLineChart';
import { IPromParams } from '@/services/prometheus-query';
import { IPromQuery } from '@/services/prometheus-config-charts';
import { usePromQueries } from './use-prom-queries';
import PrometheusChartHeader from './PrometheusChartHeader';
import { NumberConverter } from '@/utils/formatter';

const styles = require('./inspection-detail-style.less');

// const dumbData = [
//   [1540982900657, 23.45678],
//   [1540982930657, 12.345678],
//   [1540982960657, 21.123457],
//   [1540982990657, 33.555555],
//   [1540983020657, 1.6789769],
//   [1540983050657, 0],
//   [1540983080657, 12.3432543],
//   [1540983110657, 46.4356546],
//   [1540983140657, 11.546345657],
//   [1540983170657, 22.111111],
//   [1540983200657, 11.11111],
// ];
// const dumbLables = ['timestamp', 'qps'];

interface PrometheusChartProps {
  title: string;

  promQueries: IPromQuery[];
  promParams: IPromParams;
  valConverter: NumberConverter;
}

function PrometheusChart({ title, promQueries, promParams, valConverter }: PrometheusChartProps) {
  const [loading, chartLabels, oriChartData] = usePromQueries(promQueries, promParams);

  return (
    <div className={styles.chart_container}>
      <PrometheusChartHeader title={title} promQueries={promQueries} />

      {loading && <p style={{ textAlign: 'center' }}>loading...</p>}
      {!loading && oriChartData.length === 0 && <p style={{ textAlign: 'center' }}>No Data</p>}
      {!loading && oriChartData.length > 0 && (
        <div style={{ height: 200 }}>
          <SerialLineChart data={oriChartData} labels={chartLabels} valConverter={valConverter} />
        </div>
      )}
    </div>
  );
}

export default PrometheusChart;
