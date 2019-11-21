import React, { useMemo } from 'react';
import { Table } from 'antd';
import { IPromParams, IPromQuery } from '@/services/prom-query';
import { usePromQueries } from './use-prom-queries';
import PromChartHeader from './PromChartHeader';
import { IPromConfigYaxis, genValueConverter } from '@/services/prom-panel-config';

interface PromTableProps {
  title: string;

  tableColumns: [string, string];

  promQueries: IPromQuery[];
  promParams: IPromParams;
  valUnit: IPromConfigYaxis;
}

function PromTable({ title, tableColumns, promQueries, promParams, valUnit }: PromTableProps) {
  const [loading, chartLabels, oriChartData] = usePromQueries(promQueries, promParams);
  const valConverter = useMemo(() => genValueConverter(valUnit), []);

  // input:
  // data:   [[1540982900657, 10, 12, 15], [1540982900657, 10, 12, 15]]
  // labels: ['timestampe', 'foo', 'bar', 'foo']
  // output:
  // [
  //   {label: 'foo', val: '10', key: '1'},
  //   {label: 'bar', val: '12', key: '2'},
  //   {label: 'foo', val: '15', key: '3'},
  // ]
  function genDataSource() {
    if (chartLabels.length < 2 || oriChartData.length === 0) {
      return [];
    }
    return chartLabels.slice(1).map((label, index) => ({
      label,
      val: valConverter(oriChartData[0][index + 1]),
      key: `${index}`,
    }));
  }
  const dataSource: any[] = genDataSource();

  // input:
  // tableColumns: ['Host', 'CPU Num']
  // output:
  // [
  //   { title: 'Host',    dataIndex: 'label', key: 'label' },
  //   { title: 'CPU Num', dataIndex: 'val',   key: 'val' },
  // ]
  const columns = tableColumns.map((column, index) => ({
    title: column,
    dataIndex: index === 0 ? 'label' : 'val',
    key: index === 0 ? 'label' : 'val',
  }));

  return (
    <div>
      <PromChartHeader title={title} promQueries={promQueries} />

      {loading && <p style={{ textAlign: 'center' }}>loading...</p>}
      {!loading && dataSource.length === 0 && <p style={{ textAlign: 'center' }}>No Data</p>}
      {!loading && dataSource.length > 0 && (
        <Table dataSource={dataSource} columns={columns} pagination={false} />
      )}
    </div>
  );
}

export default PromTable;
