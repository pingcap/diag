const Mock = require('mockjs');

const mockedLog = {
  time: '@datetime',
  instance_name: '@name',
  'level|1': ['DEBUG', 'INFO', 'WARNING', 'ERROR', 'OTHERS'],
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

const getLogFiles = (req: any, res: any) => {
  setTimeout(() => {
    res.send(
      Mock.mock({
        'data|5-10': [
          {
            uuid: '@guid',
            file: '@word',
            filename() {
              return `${this.file}.tar.gz`;
            },
          },
        ],
      }).data,
    );
  }, 1000);
};

const searchLogs = (req: any, res: any) => {
  setTimeout(() => {
    res.send(
      Mock.mock({
        token: '@guid',
        'logs|15': [mockedLog],
      }),
    );
  }, 1000);
};

const uploadLogFile = (req: any, res: any) => {
  setTimeout(() => {
    res.send(
      Mock.mock({
        uuid: '@guid',
        file: '@word',
        filename() {
          return `${this.file}.tar.gz`;
        },
      }),
    );
  }, 1000);
};

export default {
  // for admin
  'GET /api/v1/loginstances': getLogInstances,
  'GET /api/v1/loginstances/:id/logs': searchLogs,

  // for dba
  'POST /api/v1/logfiles': uploadLogFile,
  'GET /api/v1/logfiles': getLogFiles,
  'GET /api/v1/logfiles/:id/logs': searchLogs,
};
