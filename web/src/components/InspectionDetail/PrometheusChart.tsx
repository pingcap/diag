import React, { useState, useEffect } from 'react';
import SerialLineChart from '../Chart/SerialLineChart';
import { prometheusRangeQuery, IPromParams, IMetric } from '@/services/prometheus';

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
  title?: string;

  promMetrics: IMetric[];
  promParams: IPromParams;
}

function PrometheusChart({ title, promMetrics, promParams }: PrometheusChartProps) {
  const [chartLabels, setChartLabels] = useState<string[]>([]);
  const [oriChartData, setOriChartData] = useState<number[][]>([]);

  useEffect(() => {
    function query() {
      Promise.all(
        promMetrics.map(metric =>
          prometheusRangeQuery(metric.promQL, metric.labelTemplate, promParams),
        ),
      ).then(results => {
        let labels: string[] = [];
        let data: number[][] = [];
        results.forEach((result, idx) => {
          if (idx === 0) {
            labels = result.metricLabels;
            data = result.metricValues;
          } else {
            labels = labels.concat(result.metricLabels.slice(1));
            data = data.map((item, index) => item.concat(result.metricValues[index].slice(1)));
          }
        });
        setChartLabels(labels);
        setOriChartData(data);
      });
    }

    query();
  }, [promMetrics, promParams]);

  return (
    <div>
      {title && <h4 style={{ textAlign: 'center', marginTop: 10 }}>{title}</h4>}

      {oriChartData.length > 0 ? (
        <div style={{ height: 200 }}>
          <SerialLineChart
            data={oriChartData}
            labels={chartLabels}
            valConverter={promMetrics[0].valConverter}
          />
        </div>
      ) : (
        <p style={{ textAlign: 'center' }}>No Data</p>
      )}
    </div>
  );
}

export default PrometheusChart;
