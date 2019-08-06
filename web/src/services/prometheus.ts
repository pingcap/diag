import request from '@/utils/request';

// ////
const RAW_PROM_SQLS = {
  // Overview
  vcores: 'count(node_cpu{mode="user", inspectionid="INSPECTION_ID_PLACEHOLDER"}) by (instance)',
  memory: 'node_memory_MemTotal{inspectionid="INSPECTION_ID_PLACEHOLDER"}',
  cpu_usage:
    '100 - avg by (instance) (irate(node_cpu{mode="idle", inspectionid="INSPECTION_ID_PLACEHOLDER"}[1m]) ) * 100',
  load: 'node_load1{inspectionid="INSPECTION_ID_PLACEHOLDER"}',
  memory_available: 'node_memory_MemAvailable{inspectionid="INSPECTION_ID_PLACEHOLDER"}',

  network_traffic_receive:
    'irate(node_network_receive_bytes{device!="lo", inspectionid="INSPECTION_ID_PLACEHOLDER"}[5m]) * 8',
  network_traffic_transmit:
    'irate(node_network_transmit_bytes{device!="lo", inspectionid="INSPECTION_ID_PLACEHOLDER"}[5m]) * 8',

  tcp_retrans_syn:
    'irate(node_netstat_TcpExt_TCPSynRetrans{inspectionid="INSPECTION_ID_PLACEHOLDER"}[1m])',
  tcp_retrans_slow_start:
    'irate(node_netstat_TcpExt_TCPSlowStartRetrans{inspectionid="INSPECTION_ID_PLACEHOLDER"}[1m])',
  tcp_retrans_forward:
    'irate(node_netstat_TcpExt_TCPForwardRetrans{inspectionid="INSPECTION_ID_PLACEHOLDER"}[1m])',

  io_util: 'rate(node_disk_io_time_ms{inspectionid="INSPECTION_ID_PLACEHOLDER"}[1m]) / 1000',

  // pd
  // pd cluster
  // store status
  disconnect_stores:
    'sum(pd_cluster_status{type="store_disconnected_count", inspectionid="INSPECTION_ID_PLACEHOLDER"})',
  unhealth_stores:
    'sum(pd_cluster_status{type="store_unhealth_count", inspectionid="INSPECTION_ID_PLACEHOLDER"})',
  low_space_stores:
    'sum(pd_cluster_status{type="store_low_space_count", inspectionid="INSPECTION_ID_PLACEHOLDER"})',
  down_stores:
    'sum(pd_cluster_status{type="store_down_count", inspectionid="INSPECTION_ID_PLACEHOLDER"})',
  offline_stores:
    'sum(pd_cluster_status{type="store_offline_count", inspectionid="INSPECTION_ID_PLACEHOLDER"})',
  tombstone_stores:
    'sum(pd_cluster_status{type="store_tombstone_count", inspectionid="INSPECTION_ID_PLACEHOLDER"})',
};

export const PROM_SQLS = {
  ...RAW_PROM_SQLS,

  network_traffic: [RAW_PROM_SQLS.network_traffic_receive, RAW_PROM_SQLS.network_traffic_transmit],
  tcp_retrans: [
    RAW_PROM_SQLS.tcp_retrans_syn,
    RAW_PROM_SQLS.tcp_retrans_slow_start,
    RAW_PROM_SQLS.tcp_retrans_forward,
  ],
  stores_status: [
    RAW_PROM_SQLS.disconnect_stores,
    RAW_PROM_SQLS.unhealth_stores,
    RAW_PROM_SQLS.low_space_stores,
    RAW_PROM_SQLS.down_stores,
    RAW_PROM_SQLS.offline_stores,
    RAW_PROM_SQLS.tombstone_stores,
  ],
};

export function fillInspectionId(oriPromSQL: string | string[], inspectionId: string): string[] {
  let promSQLs: string[];
  if (typeof oriPromSQL === 'string') {
    promSQLs = [oriPromSQL];
  } else {
    promSQLs = oriPromSQL;
  }
  return promSQLs.map(item => item.replace('INSPECTION_ID_PLACEHOLDER', inspectionId));
}

// ////

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
export async function prometheusRangeQuery(
  query: string,
  otherParmas: IPromParams,
): Promise<{ metricLabels: string[]; metricValues: number[][] }> {
  const params = {
    query,
    ...otherParmas,
  };
  const res = await request('/metric/query_range', { params });

  const metricLabels: string[] = ['timestamp'];
  const metricValues: number[][] = [];

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
  return { metricLabels, metricValues };
}
