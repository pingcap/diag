const Mock = require('mockjs');

const mockedLog = {
  time: '@datetime',
  filename: '@word',
  file() {
    return `${this.filename}.log`;
  },
  content: '@sentence',
};

const getLogInstances = (req: any, res: any) => {
  setTimeout(() => {
    res.send(
      Mock.mock({
        'data|5-10': [
          {
            uuid: '@guid',
            name: '@name',
          },
        ],
      }).data,
    );
  }, 1000);
};

const getLogs = (req: any, res: any) => {
  setTimeout(() => {
    res.send(
      Mock.mock({
        token: '@guid',
        'logs|15': [mockedLog],
      }),
    );
  }, 1000);
};

export default {
  'GET /api/v1/loginstances': getLogInstances,
  'GET /api/v1/loginstances/:id/logs': getLogs,
};
