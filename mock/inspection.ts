const Mock = require('mockjs');

export default {
  'GET /api/v1/instances': [
    {
      uuid: '111',
      name: 'test-cluster',
      pd: '100.0.0.0:1000, 100.0.0.1:1001, 100.0.0.2:1002',
      create_time: '2019-07-15T06:24:51Z',
      status: 'exception',
      message: 'Failed to connect to the host via ssh',
    },
    {
      uuid: '222',
      name: 'test-cluster',
      pd: '100.0.0.0:1000, 100.0.0.1:1001, 100.0.0.2:1002',
      create_time: '2019-07-15T06:24:51Z',
      status: 'exception',
      message: 'Failed to connect to the host via ssh',
    },
    {
      uuid: '333',
      name: 'test-cluster',
      pd: '100.0.0.0:1000, 100.0.0.1:1001, 100.0.0.2:1002',
      create_time: '2019-07-15T06:24:51Z',
      status: 'exception',
      message: 'Failed to connect to the host via ssh',
    },
    {
      uuid: '444',
      name: 'test-cluster',
      pd: '100.0.0.0:1000, 100.0.0.1:1001, 100.0.0.2:1002',
      create_time: '2019-07-15T06:24:51Z',
      status: 'running',
      message: '',
    },
    {
      uuid: '555',
      name: 'test-cluster',
      pd: '100.0.0.0:1000, 100.0.0.1:1001, 100.0.0.2:1002',
      create_time: '2019-07-15T06:24:51Z',
      status: 'running',
      message: '',
    },
    {
      uuid: '666',
      name: 'test-cluster',
      pd: '100.0.0.0:1000, 100.0.0.1:1001, 100.0.0.2:1002',
      create_time: '2019-07-15T06:24:51Z',
      status: 'running',
      message: '',
    },
  ],
  'POST /api/v1/instances': (req: any, res: any) => {
    res.send({
      uuid: `new-instances-${Date.now()}`,
      name: `test-cluster-${Date.now()}`,
      pd: '100.0.0.0:1000, 100.0.0.1:1001, 100.0.0.2:1002',
      create_time: new Date().toISOString(),
      status: 'running',
      message: '',
    });
  },
  'DELETE /api/v1/instances/:id': (req: any, res: any) => {
    res.status(204).send();
  },
  'GET /api/v1/instances/:id/config': (req: any, res: any) => {
    setTimeout(() => {
      res.send({
        instance_uuid: 'dc4e53ea-f5f3-49ba-a2b8-30d4d3e9d3f9',
        collect_hardware_info: true, // 硬件信息
        collect_software_info: false, // 软件信息

        collect_log: false, // 应用日志信息
        collect_log_duration: 60, // 应用日志信息时长

        collect_metric_duration: 60, // 性能监控信息时长

        collect_demsg: false, // 机器 demsg 信息

        auto_sched_duration: '', // ??
        auto_sched_start: '01:00', // 开始时间
        report_keep_duration: 0, // 保存时长
      });
    }, 1000);
  },
  'POST /api/v1/instances/:id/config': (req: any, res: any) => {
    setTimeout(() => {
      res.status(204).send();
    }, 1000);
  },
  // ////////////////
  'GET /api/v1/instances/:id/inspections': (req: any, res: any) => {
    setTimeout(() => {
      res.send(
        Mock.mock({
          'total|100-200': 10,
          'data|10': [
            {
              uuid: /\w{12}/,
              instance_id: /\w{12}/,
              'status|1': ['running', 'finish'],
              'type|1': ['manual', 'auto'],
              create_time: '@datetime',
              finish_time: '@datetime',
              report_path: '@url',
              instance_name: '@name',
            },
          ],
        }),
      );
    }, 1000);
  },
  'DELETE /api/v1/inspections/:id': (req: any, res: any) => {
    res.status(204).send();
  },
  'POST /api/v1/inspections': (req: any, res: any) => {
    setTimeout(() => {
      res.send(
        Mock.mock({
          uuid: /\w{12}/,
          instance_id: /\w{12}/,
          status: 'running',
          'type|1': 'manual',
          create_time: '@datetime',
          finish_time: '@datetime',
          report_path: '@url',
          instance_name: '@name',
        }),
      );
    }, 1000);
  },
};
