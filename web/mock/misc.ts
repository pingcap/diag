const Mock = require('mockjs');

const mockedFlamegraph = {
  uuid: '@guid',
  // ip: '@ip',
  // 'port|1-65535': 1,
  // machine(): string {
  //   return `${this.ip}:${this.port}`;
  // },
  machine: '@name',
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
    setTimeout(() => {
      res.send(Mock.mock(mockedFlamegraph));
    }, 1000);
  },
  'GET /api/v1/flamegraphs/:id': (req: any, res: any) => {
    setTimeout(() => {
      res.send(
        Mock.mock({
          image_url:
            'https://camo.githubusercontent.com/789f18134b375f4ef0ce667012aa7992bef365d5/687474703a2f2f7777772e6272656e64616e67726567672e636f6d2f466c616d654772617068732f6370752d626173682d666c616d6567726170682e737667',
          svg_url: 'http://www.brendangregg.com/FlameGraphs/cpu-bash-flamegraph.svg',
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
    setTimeout(() => {
      res.send(Mock.mock(mockedPerfProfile));
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
