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
import { getValueFormat } from 'value-formats';
import { IPromConfigYaxis } from '@/services/promtheus-panel-config';

const DEF_COLORS: string[] = '#E79FD5,#B3AD9E,#89DAC1,#17B8BE,#4DC19C,#88572C,#DDB27C,#19CDD7,#FF9833,#79C7E3,#12939A'.split(
  ',',
);

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

interface ISerailLineChartProps {
  data: number[][];
  labels: string[];

  timeFormat?: string;
  yaxis: IPromConfigYaxis;
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

function genNumberConverter(yaxis: IPromConfigYaxis) {
  const formatFunc = getValueFormat(yaxis.format);
  const valConverter = (val: number): string => {
    let { decimals } = yaxis;
    if (decimals === undefined) {
      decimals = 2;
    }
    return formatFunc(val, decimals);
  };
  return valConverter;
}

function SerialLineChart({ labels, data, timeFormat = 'HH:mm:ss', yaxis }: ISerailLineChartProps) {
  const chartLabels: string[] = useMemo(() => uniqLabels(labels), [labels]);
  const chartData = useMemo(() => convertChartData(data, chartLabels, timeFormat), [
    chartLabels,
    data,
    timeFormat,
  ]);
  const shuffedColors: string[] = useMemo(() => _.shuffle(DEF_COLORS), []);
  const valConverter = useMemo(() => genNumberConverter(yaxis), []);

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
        <YAxis width={80} type="number" tickFormatter={valConverter} />

        <CartesianGrid strokeDasharray="3 3" />
        <Tooltip
          formatter={val => valConverter(val as number)}
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
