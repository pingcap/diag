import _ from 'lodash';
import request from '@/utils/request';
import {
  bytesSizeFormatter,
  NumberConverer,
  toPercentUnit,
  toFixed,
  networkBitSizeFormatter,
  toFixed2,
  toPercent,
  toAnyUnit,
} from '@/utils/formatter';

export interface IRawMetric {
  promQLTemplate: string;
  labelTemplate: string;
  valConverter?: NumberConverer;
}

export interface IMetric {
  promQL: string;
  labelTemplate: string;
  valConverter?: NumberConverer;
}

// https://www.lodashjs.com/docs/latest#_templatestring-options
// 使用自定义的模板分隔符
// _.templateSettings.interpolate = /{{([\s\S]+?)}}/g;
// var compiled = _.template('hello {{ user }}!');
// compiled({ 'user': 'mustache' });
// // => 'hello mustache!'

_.templateSettings.interpolate = /{{([\s\S]+?)}}/g;

// ////
const RAW_METRICS: { [key: string]: IRawMetric } = {
  // ////////////////////////////
  // Overview
  vcores: {
    promQLTemplate: 'count(node_cpu{mode="user", inspectionid="{{inspectionId}}"}) by (instance)',
    labelTemplate: '{{instance}}',
  },

  memory: {
    promQLTemplate: 'node_memory_MemTotal{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{instance}}',
    valConverter: bytesSizeFormatter,
  },

  cpu_usage: {
    promQLTemplate:
      '100 - avg by (instance) (irate(node_cpu{mode="idle", inspectionid="{{inspectionId}}"}[1m]) ) * 100',
    labelTemplate: '{{instance}}',
    valConverter: val => toPercentUnit(val, 1),
  },
  load: {
    promQLTemplate: 'node_load1{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{instance}}',
    valConverter: val => toFixed(val, 1),
  },
  memory_available: {
    promQLTemplate: 'node_memory_MemAvailable{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{instance}}',
    valConverter: bytesSizeFormatter,
  },

  network_traffic_receive: {
    promQLTemplate:
      'irate(node_network_receive_bytes{device!="lo", inspectionid="{{inspectionId}}"}[5m]) * 8',
    labelTemplate: 'Inbound: {{instance}}',
    valConverter: networkBitSizeFormatter,
  },
  network_traffic_transmit: {
    promQLTemplate:
      'irate(node_network_transmit_bytes{device!="lo", inspectionid="{{inspectionId}}"}[5m]) * 8',
    labelTemplate: 'Outbound: {{instance}}',
    valConverter: networkBitSizeFormatter,
  },

  tcp_retrans_syn: {
    promQLTemplate: 'irate(node_netstat_TcpExt_TCPSynRetrans{inspectionid="{{inspectionId}}"}[1m])',
    labelTemplate: '{{instance}} - TCPSynRetrans',
    valConverter: toFixed2,
  },
  tcp_retrans_slow_start: {
    promQLTemplate:
      'irate(node_netstat_TcpExt_TCPSlowStartRetrans{inspectionid="{{inspectionId}}"}[1m])',
    labelTemplate: '{{instance}} - TCPSlowStartRetrans',
    valConverter: toFixed2,
  },
  tcp_retrans_forward: {
    promQLTemplate:
      'irate(node_netstat_TcpExt_TCPForwardRetrans{inspectionid="{{inspectionId}}"}[1m])',
    labelTemplate: '{{instance}} - TCPForwardRetrans',
    valConverter: toFixed2,
  },

  io_util: {
    promQLTemplate: 'rate(node_disk_io_time_ms{inspectionid="{{inspectionId}}"}[1m]) / 1000',
    labelTemplate: '{{instance}} - {{device}}',
    valConverter: val => toPercent(val, 4),
  },

  // ////////////////////////////
  // PD
  // pd cluster
  disconnect_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_disconnected_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'Disconnect Stores',
  },
  unhealth_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_unhealth_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'Unhealth Stores',
  },
  low_space_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_low_space_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'LowSpace Stores',
  },
  down_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_down_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'Down Stores',
  },
  offline_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_offline_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'Offline Stores',
  },
  tombstone_stores: {
    promQLTemplate:
      'sum(pd_cluster_status{type="store_tombstone_count", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'Tombstone Stores',
  },

  storage_capacity: {
    promQLTemplate:
      'sum(pd_cluster_status{type="storage_capacity", inspectionid="{{inspectionId}}" })',
    labelTemplate: 'storage capacity',
    valConverter: bytesSizeFormatter,
  },

  storage_size: {
    promQLTemplate: 'pd_cluster_status{type="storage_size"}',
    labelTemplate: 'storage size',
    valConverter: val => bytesSizeFormatter(val, true, 2),
  },
  storage_size_ratio: {
    promQLTemplate:
      'avg(pd_cluster_status{type="storage_size", inspectionid="{{inspectionId}}"}) / avg(pd_cluster_status{type="storage_capacity", inspectionid="{{inspectionId}}"})',
    labelTemplate: 'used ratio',
    valConverter: val => toPercent(val, 6),
  },

  regions_label_level: {
    promQLTemplate: 'pd_regions_label_level{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{type}}',
  },

  regions_status: {
    promQLTemplate: 'pd_regions_status{inspectionid="{{inspectionId}}"}',
    labelTemplate: '{{type}}',
  },
  regions_status_sum: {
    promQLTemplate: 'sum(pd_regions_status) by (instance, type)',
    labelTemplate: '{{type}}',
  },

  // balance
  store_available: {
    promQLTemplate: '{inspectionid="{{inspectionId}}", type="store_available"}',
    labelTemplate: 'store-{{store}}',
    valConverter: val => bytesSizeFormatter(val, true, 2),
  },
  store_leader_score: {
    promQLTemplate:
      'pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="leader_score"}',
    labelTemplate: 'tikv-{{store}}',
  },
  store_region_score: {
    promQLTemplate:
      'pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="region_score"}',
    labelTemplate: 'tikv-{{store}}',
  },
  store_leader_count: {
    promQLTemplate:
      'pd_scheduler_store_status{inspectionid="{{inspectionId}}", type="leader_count"}',
    labelTemplate: 'tikv-{{store}}',
  },

  // hot region
  hot_write_region_leader_distribution: {
    promQLTemplate:
      'pd_hotspot_status{inspectionid="{{inspectionId}}",type="hot_write_region_as_leader"}',
    labelTemplate: '{{store}}',
  },
  hot_write_region_peer_distribution: {
    promQLTemplate:
      'pd_hotspot_status{inspectionid="{{inspectionId}}",type="hot_write_region_as_peer"}',
    labelTemplate: '{{store}}',
  },
  hot_read_region_leader_distribution: {
    promQLTemplate:
      'pd_hotspot_status{inspectionid="{{inspectionId}}",type="hot_read_region_as_leader"}',
    labelTemplate: '{{store}}',
  },

  // operator
  schedule_operator_create: {
    promQLTemplate:
      'sum(delta(pd_schedule_operators_count{inspectionid="{{inspectionId}}", event="create"}[1m])) by (type)',
    labelTemplate: '{{type}}',
  },
  schedule_operator_timeout: {
    promQLTemplate:
      'sum(delta(pd_schedule_operators_count{inspectionid="{{inspectionId}}", event="timeout"}[1m])) by (type)',
    labelTemplate: '{{type}}',
  },

  // etcd
  handle_txn_count: {
    promQLTemplate:
      'sum(rate(pd_txn_handle_txns_duration_seconds_count[5m])) by (instance, result)',
    labelTemplate: '{{instance}} : {{result}}',
    valConverter: toFixed2,
  },
  wal_fsync_duration_seconds_99: {
    promQLTemplate:
      'histogram_quantile(0.99, sum(rate(etcd_disk_wal_fsync_duration_seconds_bucket[5m])) by (instance, le))',
    labelTemplate: '{{instance}}',
    valConverter: val => toAnyUnit(val, 1000 * 1000, 2, 'us'),
  },

  // tidb
  handle_request_duration_seconds_bucket: {
    promQLTemplate:
      'histogram_quantile(0.98, sum(rate(pd_client_request_handle_requests_duration_seconds_bucket[30s])) by (type, le))',
    labelTemplate: '{{type}} 98th percentile',
    valConverter: val => toAnyUnit(val, 1000 * 1000, 2, 'us'),
  },
  handle_request_duration_seconds_avg: {
    promQLTemplate:
      'avg(rate(pd_client_request_handle_requests_duration_seconds_sum[30s])) by (type) /  avg(rate(pd_client_request_handle_requests_duration_seconds_count[30s])) by (type)',
    labelTemplate: '{{type}} average',
    valConverter: val => toAnyUnit(val, 1000 * 1000, 2, 'us'),
  },

  // heartbeat
  region_heartbeat_latency_99: {
    promQLTemplate:
      'round(histogram_quantile(0.99, sum(rate(pd_scheduler_region_heartbeat_latency_seconds_bucket[5m])) by (store, le)), 1000)',
    labelTemplate: 'store{{store}}',
    valConverter: val => toAnyUnit(val, 1000, 1, 'ms'),
  },
};

export const RAW_METRICS_ARR: { [key: string]: IRawMetric[] } = {
  ...Object.keys(RAW_METRICS).reduce((accu, curVal) => {
    accu[curVal] = [RAW_METRICS[curVal]];
    return accu;
  }, {}),

  network_traffic: [RAW_METRICS.network_traffic_receive, RAW_METRICS.network_traffic_transmit],
  tcp_retrans: [
    RAW_METRICS.tcp_retrans_syn,
    RAW_METRICS.tcp_retrans_slow_start,
    RAW_METRICS.tcp_retrans_forward,
  ],
  stores_status: [
    RAW_METRICS.disconnect_stores,
    RAW_METRICS.unhealth_stores,
    RAW_METRICS.low_space_stores,
    RAW_METRICS.down_stores,
    RAW_METRICS.offline_stores,
    RAW_METRICS.tombstone_stores,
  ],
  region_health: [RAW_METRICS.regions_status, RAW_METRICS.regions_status_sum],

  handle_request_duration_seconds: [
    RAW_METRICS.handle_request_duration_seconds_bucket,
    RAW_METRICS.handle_request_duration_seconds_avg,
  ],
};

export function fillPromQLTemplate(rawMetrics: IRawMetric[], inspectionId: string): IMetric[] {
  return rawMetrics.map(rawMetric => ({
    ...rawMetric,
    promQL: _.template(rawMetric.promQLTemplate)({ inspectionId }),
  }));
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
        metricValues[arrIdx][idx + 1] = +arr[1]; // convert string to number
      });
    });
  }
  return { metricLabels, metricValues };
}
