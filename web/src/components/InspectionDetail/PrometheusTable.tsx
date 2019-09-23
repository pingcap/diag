import React from 'react';
import { Table } from 'antd';
import { IPromParams } from '@/services/prometheus-query';
import { IPromQuery } from '@/services/prometheus-config-charts';
import { usePromQueries } from './use-prom-queries';
import PrometheusChartHeader from './PrometheusChartHeader';

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
