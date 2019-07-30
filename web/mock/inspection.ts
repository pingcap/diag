const Mock = require('mockjs');

const mockedInstance = {
  uuid: /\w{12}/,
  name: '@name',
  user: '@first',
  ip: '@ip',
  'port|1-65535': 1,
  pd(): string {
    return `${this.ip}:${this.port}`;
  },
  create_time: '@datetime',
  'status|1': ['pending', 'exception', 'success'],
  message: '@title',
};

const mockedInspection = {
  uuid: '@guid',
  instance_id: '@guid',
  instance_name: '@name',
  user: '@first',
  'status|1': ['running', 'exception', 'success'],
  'type|1': ['manual', 'auto'],
  create_time: '@datetime',
  finish_time: '@datetime',
};

export default {
  'GET /api/v1/instances': (req: any, res: any) => {
    setTimeout(() => {
      res.send(
        Mock.mock({
          'data|10': [mockedInstance],
        }).data,
      );
    }, 1000);
  },
  'POST /api/v1/instances': (req: any, res: any) => {
    setTimeout(() => {
      res.send(Mock.mock(mockedInstance));
    }, 1000);
  },
  'DELETE /api/v1/instances/:id': (req: any, res: any) => {
    setTimeout(() => {
      res.status(204).send();
    }, 1000);
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
          'data|10': [mockedInspection],
        }),
      );
    }, 1000);
  },
  'GET /api/v1/inspections': (req: any, res: any) => {
    setTimeout(() => {
      res.send(
        Mock.mock({
          'total|100-200': 10,
          'data|10': [mockedInspection],
        }),
      );
    }, 1000);
  },
  'DELETE /api/v1/inspections/:id': (req: any, res: any) => {
    setTimeout(() => {
      res.status(204).send();
    }, 1000);
  },
  'PUT /api/v1/inspections/:id': (req: any, res: any) => {
    setTimeout(() => {
      res.status(204).send();
    }, 1000);
  },
  'POST /api/v1/instances/:id/inspections': (req: any, res: any) => {
    setTimeout(() => {
      res.send(Mock.mock(mockedInspection));
    }, 1000);
  },
};
