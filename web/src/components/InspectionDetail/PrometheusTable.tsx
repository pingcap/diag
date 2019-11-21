import React from 'react';
import { Table } from 'antd';
import { IPromParams } from '@/services/prometheus-query';
import { IPromQuery } from '@/services/prometheus-config-charts';
import { usePromQueries } from './use-prom-queries';
import PrometheusChartHeader from './PrometheusChartHeader';

interface PrometheusTableProps {
  title: string;

  tableColumns: [string, string];

  promQueries: IPromQuery[];
  promParams: IPromParams;
}

function PrometheusTable({ title, tableColumns, promQueries, promParams }: PrometheusTableProps) {
  const [loading, chartLabels, oriChartData] = usePromQueries(promQueries, promParams);

  // can replace it by useMemo
  function genDataSource() {
    if (chartLabels.length < 2 || oriChartData.length === 0) {
      return [];
    }
    return chartLabels.slice(1).map((label, index) => ({
      label,
      val: promQueries[0].valConverter
        ? promQueries[0].valConverter(oriChartData[0][index + 1])
        : oriChartData[0][index + 1],
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
      <PrometheusChartHeader title={title} promQueries={promQueries} />

      {loading && <p style={{ textAlign: 'center' }}>loading...</p>}
      {!loading && dataSource.length === 0 && <p style={{ textAlign: 'center' }}>No Data</p>}
      {!loading && dataSource.length > 0 && (
        <Table dataSource={dataSource} columns={columns} pagination={false} />
      )}
    </div>
  );
}

export default PrometheusTable;
