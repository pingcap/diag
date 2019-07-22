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
              report_path: '@url',
            },
          ],
        }),
      );
    }, 1000);
  },
};
