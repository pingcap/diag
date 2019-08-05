import request from '@/utils/request';

// ////

export const VCORES_PROM_SQL =
  'count(node_cpu{mode="user", inspectionid="INSPECTION_ID_PLACEHOLDER"}) by (instance)';

export function fillInspectionId(oriPromSQL: string, inspectionId: string) {
  return oriPromSQL.replace('INSPECTION_ID_PLACEHOLDER', inspectionId);
}

// ////

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
export async function prometheusRangeQuery(
  query: string,
  start: number,
  end: number,
  sampleCount: number = 15,
): Promise<[string[], any[]]> {
  const step = Math.floor((end - start) / sampleCount);
  const params = {
    query,
    start,
    end,
    step,
  };
  const res = await request('/metric/query_range', { params });

  const metricLabels: string[] = ['timestamp'];
  const metricValues: any[] = [];

  if (res !== undefined) {
    const { data } = res;
    const metricValuesItemArrLength = data.result.length + 1;
    data.result.forEach((item: any, idx: number) => {
      const label = item.metric.instance || `instance-${idx + 1}`;

      metricLabels.push(label);
      // item.values is a 2 demension array
      const curValuesArr = item.values;
      curValuesArr.forEach((arr: any[], arrIdx: number) => {
        if (metricValues[arrIdx] === undefined) {
          metricValues[arrIdx] = Array(metricValuesItemArrLength).fill(0);
          metricValues[arrIdx][0] = arr[0] * 1000; // convert seconds to milliseconds
        }
        metricValues[arrIdx][idx + 1] = +arr[1]; // convert string to number
      });
    });
  }
  return [metricLabels, metricValues];
}
