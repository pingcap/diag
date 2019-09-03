import React, { useMemo } from 'react';
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
  Legend,
} from 'recharts';
import moment from 'moment';
import _ from 'lodash';
import { NumberConverer } from '@/utils/formatter';

// const styles = require('./Chart.less');

const DEF_COLORS: string[] = '#E79FD5,#B3AD9E,#89DAC1,#17B8BE,#4DC19C,#88572C,#DDB27C,#19CDD7,#FF9833,#79C7E3,#12939A'.split(
  ',',
);

interface ISerailLineChartProps {
  style?: object;

  data: number[][];
  labels: string[];

  timeFormat?: string;
  valConverter?: NumberConverer;
}

function convertChartData(oriData: number[][], chartLabels: string[], timeFormat: string) {
  return oriData.map(d => {
    const obj = {};
    chartLabels.forEach((cl, idx) => {
      if (idx === 0) {
        obj[cl] = moment(d[idx]).format(timeFormat);
      } else {
        obj[cl] = d[idx];
      }
    });
    return obj;
  });
}

function loopGenUniqName(
  oriName: string,
  existNames: string[],
  tryCount: number = 1,
): [string, number] {
  const newName: string = tryCount > 1 ? `${oriName}-${tryCount}` : oriName;
  if (!existNames.includes(newName)) {
    return [newName, tryCount];
  }
  return loopGenUniqName(oriName, existNames, tryCount + 1);
}

function uniqLabels(oriLabels: string[]): string[] {
  let newLabels: string[] = [];
  const duplicatedLabels: string[] = [];
  oriLabels.forEach(oriLabel => {
    const [newLabel, tryCount] = loopGenUniqName(oriLabel, newLabels);
    if (tryCount === 2) {
      duplicatedLabels.push(oriLabel);
    }
    newLabels.push(newLabel);
  });
  if (duplicatedLabels.length > 0) {
    newLabels = newLabels.map(label => (duplicatedLabels.includes(label) ? `${label}-1` : label));
  }
  return newLabels;
}

function SerialLineChart({
  labels,
  data,
  timeFormat = 'HH:mm:ss',
  valConverter,
}: ISerailLineChartProps) {
  const chartLabels: string[] = useMemo(() => uniqLabels(labels), [labels]);
  const chartData = useMemo(() => convertChartData(data, chartLabels, timeFormat), [
    chartLabels,
    data,
    timeFormat,
  ]);
  const shuffedColors: string[] = useMemo(() => _.shuffle(DEF_COLORS), []);

  return (
    <ResponsiveContainer width="100%" height="100%">
      <LineChart
        data={chartData}
        margin={{
          top: 5,
          right: 10,
          bottom: 0,
        }}
      >
        <XAxis dataKey={chartLabels[0]} />
        <YAxis
          width={80}
          type="number"
          tickFormatter={val => (valConverter ? valConverter(val) : val)}
        />

        <CartesianGrid strokeDasharray="3 3" />
        <Tooltip
          formatter={val => (valConverter ? valConverter(val as number) : val)}
          wrapperStyle={{
            zIndex: 1,
          }}
        />
        {/* https://github.com/recharts/recharts/issues/614 */}
        {/* Position Legend on the right side of a graph #614 */}
        <Legend
          layout="vertical"
          verticalAlign="top"
          align="right"
          wrapperStyle={{
            paddingLeft: '12px',
            maxHeight: '80%',
            overflowY: 'auto',
          }}
        />

        {chartLabels.slice(1).map((cl, idx) => (
          <Line
            key={cl}
            type="monotone"
            dataKey={cl}
            stroke={shuffedColors[idx]}
            activeDot={{ r: 6 }}
          />
        ))}
      </LineChart>
    </ResponsiveContainer>
  );
}

export default SerialLineChart;
