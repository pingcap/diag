import React, { useMemo } from 'react';
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Legend,
} from 'recharts';
import { Tooltip } from 'antd';
import moment from 'moment';

// const styles = require('./Chart.less');

const colorsConfig: string[] = '#E79FD5,#B3AD9E,#89DAC1,#17B8BE,#4DC19C,#88572C,#DDB27C,#19CDD7,#FF9833,#79C7E3,#12939A'.split(
  ',',
);

interface ISerailLineChartProps {
  style?: object;

  title?: string;

  data: any[];
  labels: string[];

  showLabel?: boolean;

  timeFormat?: string;

  chartValConverter?: any;
}

function convertChartData(oriData: any[], lables: string[], timeFormat: string) {
  return oriData.map(d => {
    const obj = {};
    lables.forEach((l, idx) => {
      if (idx === 0) {
        obj[l] = moment(d[idx]).format(timeFormat);
      } else {
        obj[l] = d[idx];
      }
    });
    return obj;
  });
}

function SerialLineChart({ labels, data, timeFormat = 'HH:mm:ss' }: ISerailLineChartProps) {
  const chartLabels: string[] = labels;
  const chartData: any[] = useMemo(() => convertChartData(data, labels, timeFormat), [
    labels,
    data,
    timeFormat,
  ]);

  return (
    <ResponsiveContainer width="100%">
      <LineChart data={chartData}>
        <XAxis dataKey={chartLabels[0]} />
        <YAxis />

        <CartesianGrid strokeDasharray="3 3" />
        <Tooltip />
        <Legend />

        {chartLabels.slice(1).map((cl, idx) => (
          <Line
            key={cl}
            type="monotone"
            dataKey={cl}
            stroke={colorsConfig[idx]}
            activeDot={{ r: 6 }}
          />
        ))}
      </LineChart>
    </ResponsiveContainer>
  );
}

export default SerialLineChart;
