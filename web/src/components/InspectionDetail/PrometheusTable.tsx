import React, { useState, useEffect } from 'react';
import { Table } from 'antd';
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

interface PrometheusTableProps {
  title?: string;

  tableColumns: [string, string];

  promMetrics: IMetric[];
  promParams: IPromParams;
}

function PrometheusTable({ title, tableColumns, promMetrics, promParams }: PrometheusTableProps) {
  const [loading, setLoading] = useState(false);

  const [dataSource, setDataSource] = useState<any[]>([]);

  const columns = tableColumns.map((column, index) => ({
    title: column,
    dataIndex: index === 0 ? 'label' : 'val',
    key: index === 0 ? 'label' : 'val',
  }));

  useEffect(() => {
    function query() {
      setLoading(true);
      Promise.all(
        promMetrics.map(metric =>
          prometheusRangeQuery(metric.promQL, metric.labelTemplate, promParams),
        ),
      ).then(results => {
        let labels: string[] = [];
        let data: number[][] = [];
        results
          .filter(result => result.metricValues.length > 0)
          .forEach((result, idx) => {
            if (idx === 0) {
              labels = result.metricLabels;
              data = result.metricValues;
            } else {
              labels = labels.concat(result.metricLabels.slice(1));
              const emtpyPlacehoder: number[] = Array(result.metricLabels.length).fill(0);
              data = data.map((item, index) =>
                // the result.metricValues may have different length
                // so result.metricValues[index] may undefined
                item.concat((result.metricValues[index] || emtpyPlacehoder).slice(1)),
              );
            }
          });
        setLoading(false);

        setDataSource([
          {
            label: labels[1],
            val: promMetrics[0].valConverter ? promMetrics[0].valConverter(data[0][1]) : data[0][1],
          },
        ]);
      });
    }

    query();
  }, [promMetrics, promParams]);

  return (
    <div>
      {title && <h4 style={{ textAlign: 'center', marginTop: 10 }}>{title}</h4>}
      {loading && <p style={{ textAlign: 'center' }}>loading...</p>}
      {!loading && dataSource.length === 0 && <p style={{ textAlign: 'center' }}>No Data</p>}
      {!loading && dataSource.length > 0 && (
        <Table dataSource={dataSource} columns={columns} pagination={false} />
      )}
    </div>
  );
}

export default PrometheusTable;
