const Mock = require('mockjs');

export default {
  'GET /api/v1/flamegraphs': (req: any, res: any) => {
    setTimeout(() => {
      res.send(
        Mock.mock({
          'total|100-200': 10,
          'data|10': [
            {
              uuid: /\w{12}/,
              machine: /\d{3}\.\d{3}\.\d{3}\.\d{3}:\d{4}/,
              user: '@name',
              'status|1': ['running', 'finish'],
              create_time: '@datetime',
              finish_time: '@datetime',
            },
          ],
        }),
      );
    }, 1000);
  },
  'POST /api/v1/flamegraphs': (req: any, res: any) => {
    // console.log(req.body); // {machine: 123}
    setTimeout(() => {
      res.send(
        Mock.mock({
          uuid: /\w{12}/,
          machine: req.body.machine, // not req.params
          user: '@name',
          status: 'running',
          create_time: '@datetime',
          finish_time: '@datetime',
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
          'data|10': [
            {
              uuid: /\w{12}/,
              machine: /\d{3}\.\d{3}\.\d{3}\.\d{3}:\d{4}/,
              user: '@name',
              'status|1': ['running', 'finish'],
              create_time: '@datetime',
              finish_time: '@datetime',
            },
          ],
        }),
      );
    }, 1000);
  },
  'POST /api/v1/perfprofiles': (req: any, res: any) => {
    // console.log(req.body); // {machine: 123}
    setTimeout(() => {
      res.send(
        Mock.mock({
          uuid: /\w{12}/,
          machine: req.body.machine, // not req.params
          user: '@name',
          status: 'running',
          create_time: '@datetime',
          finish_time: '@datetime',
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
