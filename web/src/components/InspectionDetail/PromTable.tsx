import React, { useMemo } from 'react';
import { Table } from 'antd';
import { IPromParams, IPromQuery } from '@/services/prom-query';
import { usePromQueries } from './use-prom-queries';
import PromChartHeader from './PromChartHeader';
import { IPromConfigYaxis, genNumberConverter } from '@/services/prom-panel-config';

interface PromTableProps {
  title: string;

  tableColumns: [string, string];

  promQueries: IPromQuery[];
  promParams: IPromParams;
  valUnit: IPromConfigYaxis;
}

function PromTable({ title, tableColumns, promQueries, promParams, valUnit }: PromTableProps) {
  const [loading, chartLabels, oriChartData] = usePromQueries(promQueries, promParams);
  const valConverter = useMemo(() => genNumberConverter(valUnit), []);

  // can replace it by useMemo
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

  const columns = tableColumns.map((column, index) => ({
    title: column,
    dataIndex: index === 0 ? 'label' : 'val',
    key: index === 0 ? 'label' : 'val',
  }));
  const dataSource: any[] = genDataSource();

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
