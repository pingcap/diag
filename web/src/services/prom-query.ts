import _ from 'lodash';
import request from '@/utils/request';

export interface IPromQuery {
  promQL: string;
  labelTemplate: string;
}

export interface IPromParams {
  start: number;
  end: number;
  step: number;
}

// https://www.lodashjs.com/docs/latest#_templatestring-options
// 使用自定义的模板分隔符
// _.templateSettings.interpolate = /{{([\s\S]+?)}}/g;
// var compiled = _.template('hello {{ user }}!');
// compiled({ 'user': 'mustache' });
// // => 'hello mustache!'
_.templateSettings.interpolate = /{{([\s\S]+?)}}/g;

// request:
// http://localhost:8000/api/v1/metric/query_range?query=pd_cluster_status%7Btype%3D%22storage_size%22%7D&start=1560836237&end=1560836537&step=20
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
// query single promQL
// 对单个 promQL 进行查询，结果中可能包含多个 metric
export async function promRangeQuery(
  promQuery: IPromQuery,
  promParmas: IPromParams,
): Promise<{ metricLabels: string[]; metricValues: number[][] }> {
  const params = {
    query: promQuery.promQL,
    ...promParmas,
  };
  const res = await request('/metric/query_range', { params });

  const metricLabels: string[] = ['timestamp'];
  const metricValues: number[][] = [];

  if (res !== undefined) {
    const { data } = res;
    const metricValuesItemArrLength = data.result.length + 1; // plus one for "timestamp" label
    data.result.forEach((item: any, idx: number) => {
      let label = '';
      try {
        label = _.template(promQuery.labelTemplate)(item.metric);
      } catch (err) {
        console.log(err);
      }
      if (label === '') {
        label = `instance-${idx + 1}`;
      }
      metricLabels.push(label);

      // item.values is a 2 demension array
      const curValuesArr = item.values;
      curValuesArr.forEach((arr: any[], arrIdx: number) => {
        if (metricValues[arrIdx] === undefined) {
          // different metric values array may are different length
          // so some metric values array length are short than normal length
          // we need to fill it with 0
          // 不同的 metric 的 values 数组长度可以不一样，需要考虑到这种情况
          // 对缺失的数组元素先填补 0
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

// query multiple promQLs at the same time
// 同时对多个 promQL 进行查询，对结果进行聚合：合并 labels 和 values 数组
export async function promRangeQueries(promQueries: IPromQuery[], promParams: IPromParams) {
  const results = await Promise.all(
    promQueries.map(promQuery => promRangeQuery(promQuery, promParams)),
  );

  let labels: string[] = [];
  let data: number[][] = [];
  results
    .filter(result => result.metricValues.length > 0) // 过滤掉结果为空查询
    .forEach((result, idx) => {
      if (idx === 0) {
        labels = result.metricLabels;
        data = result.metricValues;
      } else {
        // 聚合时，第一列为相同的 timestamp，仅保留第一个 result 的 timestamp，其余 result 的 timestamp 抛弃
        labels = labels.concat(result.metricLabels.slice(1));
        // 不同的 promQL 查询结果中的 values 长度可能不一样，对缺失的元素进行补 0
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
