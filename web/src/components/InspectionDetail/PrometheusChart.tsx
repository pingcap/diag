import React, { useState, useEffect } from 'react';
import SerialLineChart from '../Chart/SerialLineChart';
import { prometheusRangeQuery } from '@/services/prometheus';

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
  promSQLStr: string;
}

function PrometheusChart({ promSQLStr }: PrometheusChartProps) {
  const [chartLabels, setChartLabels] = useState<string[]>([]);
  const [oriChartData, setOriChartData] = useState<any[]>([]);

  useEffect(() => {
    function query() {
      const end = Math.floor(new Date().getTime() / 1000); // convert millseconds to seconds
      const start = end - 1 * 60 * 60;
      prometheusRangeQuery(promSQLStr, start, end).then((result: [string[], any[]]) => {
        setChartLabels(result[0]);
        setOriChartData(result[1]);
      });
    }
    query();
  }, []);

  return (
    <div style={{ height: 200 }}>
      {oriChartData.length > 0 ? (
        <SerialLineChart data={oriChartData} labels={chartLabels} />
      ) : (
        'No Data'
      )}
    </div>
  );
}

export default PrometheusChart;
