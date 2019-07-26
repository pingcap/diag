const Mock = require('mockjs');

const mockedFlamegraph = {
  uuid: '@guid',
  ip: '@ip',
  'port|1-65535': 1,
  machine(): string {
    return `${this.ip}:${this.port}`;
  },
  user: '@first',
  'status|1': ['running', 'finish'],
  create_time: '@datetime',
  finish_time: '@datetime',
};

const mockedPerfProfile = mockedFlamegraph;

export default {
  'GET /api/v1/flamegraphs': (req: any, res: any) => {
    setTimeout(() => {
      res.send(
        Mock.mock({
          'total|100-200': 10,
          'data|10': [mockedFlamegraph],
        }),
      );
    }, 1000);
  },
  'POST /api/v1/flamegraphs': (req: any, res: any) => {
    // console.log(req.body); // {machine: 123}
    setTimeout(() => {
      res.send(
        Mock.mock({
          ...mockedFlamegraph,
          machine: req.body.machine, // not req.params
        }),
      );
    }, 1000);
  },
  'DELETE /api/v1/flamegraphs/:id': (req: any, res: any) => {
    setTimeout(() => {
      res.status(204).send();
    }, 1000);
  },
  'PUT /api/v1/flamegraphs/:id': (req: any, res: any) => {
    setTimeout(() => {
      res.status(204).send();
    }, 2000);
  },
  // ////////////////////////////
  'GET /api/v1/perfprofiles': (req: any, res: any) => {
    setTimeout(() => {
      res.send(
        Mock.mock({
          'total|100-200': 10,
          'data|10': [mockedPerfProfile],
        }),
      );
    }, 1000);
  },
  'POST /api/v1/perfprofiles': (req: any, res: any) => {
    // console.log(req.body); // {machine: 123}
    setTimeout(() => {
      res.send(
        Mock.mock({
          ...mockedPerfProfile,
          machine: req.body.machine, // not req.params
        }),
      );
    }, 1000);
  },
  'DELETE /api/v1/perfprofiles/:id': (req: any, res: any) => {
    setTimeout(() => {
      res.status(204).send();
    }, 1000);
  },
  'PUT /api/v1/perfprofiles/:id': (req: any, res: any) => {
    setTimeout(() => {
      res.status(204).send();
    }, 2000);
  },
};
