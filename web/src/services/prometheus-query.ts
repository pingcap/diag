import _ from 'lodash';
import request from '@/utils/request';
import { IPromQuery } from './prometheus-config-charts';

export interface IPromParams {
  start: number;
  end: number;
  step: number;
}

// request:
// http://localhost:3000/metric/api/v1/query_range?query=pd_cluster_status%7Btype%3D%22storage_size%22%7D&start=1560836237&end=1560836537&step=20
// response:
// {
//   "status": "success",
//   "data": {
//     "resultType": "matrix",
//     "result": [
//       {
//         "metric": {
//           "__name__": "pd_cluster_status",
//           "cluster": "2553e691-81de-438e-94e3-b67d39aaae52",
//           "instance": "tidb-default-pd-0",
//           "job": "tidb-cluster",
//           "kubernetes_namespace": "2553e691-81de-438e-94e3-b67d39aaae52",
//           "kubernetes_node": "172.16.4.96",
//           "kubernetes_pod_ip": "10.233.93.71",
//           "namespace": "global",
//           "type": "storage_size"
//         },
//         "values": [
//           [
//             1560836339,
//             "132132072"
//           ],
//           [
//             1560836359,
//             "132132072"
//           ],
//           ...
//         ]
//       }
//     ]
//   }
// }
// return :
// labels: ['timestamp', 'tidb-default-pd-0', 'tidb-default-pd-1']
// values: [
//   [1560836339, 132132072, 132132800],
//   [1560836359, 132132071, 132132801],
// ]
export async function promRangeQuery(
  query: string,
  labelTemplate: string,
  promParmas: IPromParams,
): Promise<{ metricLabels: string[]; metricValues: number[][] }> {
  const params = {
    query,
    ...promParmas,
  };
  const res = await request('/metric/query_range', { params });

  const metricLabels: string[] = ['timestamp'];
  const metricValues: number[][] = [];

  if (res !== undefined) {
    const { data } = res;
    const metricValuesItemArrLength = data.result.length + 1;
    data.result.forEach((item: any, idx: number) => {
      let label = '';
      try {
        label = _.template(labelTemplate)(item.metric);
      } catch (err) {
        label = `instance-${idx + 1}`;
      }

      metricLabels.push(label);
      // item.values is a 2 demension array
      const curValuesArr = item.values;
      curValuesArr.forEach((arr: any[], arrIdx: number) => {
        if (metricValues[arrIdx] === undefined) {
          metricValues[arrIdx] = Array(metricValuesItemArrLength).fill(0);
          metricValues[arrIdx][0] = arr[0] * 1000; // convert seconds to milliseconds
        }
        let numVal = +arr[1];
        if (Number.isNaN(numVal)) {
          numVal = 0;
        }
        metricValues[arrIdx][idx + 1] = numVal; // convert string to number
      });
    });
  }
  return { metricLabels, metricValues };
}

export async function promRangeQueries(promQueries: IPromQuery[], promParams: IPromParams) {
  const results = await Promise.all(
    promQueries.map(metric => promRangeQuery(metric.promQL, metric.labelTemplate, promParams)),
  );

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
  return { labels, data };
}
