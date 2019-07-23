const Mock = require('mockjs');

export default {
  'GET /api/v1/me': (req: any, res: any) => {
    setTimeout(() => {
      res.send(
        Mock.mock({
          name: 'Serati Ma',
          avatar: 'https://gw.alipayobjects.com/zos/antfincdn/XAosXuNZyF/BiazfanxmamNRoxxVxka.png',
          'username|1': ['admin', 'dba'],
          'role|1': ['admin', 'dba'],
          ka: true,
        }),
      );
    }, 1000);
  },
  'POST /api/v1/login': (req: any, res: any) => {
    const { password, username } = req.body;
    if (password === 'tidb' && (username === 'admin' || username === 'dba')) {
      setTimeout(() => {
        res.send({
          name: 'Serati Ma',
          avatar: 'https://gw.alipayobjects.com/zos/antfincdn/XAosXuNZyF/BiazfanxmamNRoxxVxka.png',
          username: req.body.username,
          role: req.body.username,
          ka: true,
        });
      }, 1000);
      return;
    }
    setTimeout(() => {
      res.status(401).send({
        status: 'login failed',
        message: "username or password doesn't match",
      });
    }, 1000);
  },
  'POST /api/v1/logout': (req: any, res: any) => {
    res.status(204).send();
  },
  'POST /api/register': (req, res) => {
    res.send({ status: 'ok', currentAuthority: 'user' });
  },
  'GET /api/500': (req, res) => {
    res.status(500).send({
      timestamp: 1513932555104,
      status: 500,
      error: 'error',
      message: 'error',
      path: '/base/category/list',
    });
  },
  'GET /api/404': (req, res) => {
    res.status(404).send({
      timestamp: 1513932643431,
      status: 404,
      error: 'Not Found',
      message: 'No message available',
      path: '/base/category/list/2121212',
    });
  },
  'GET /api/403': (req, res) => {
    res.status(403).send({
      timestamp: 1513932555104,
      status: 403,
      error: 'Unauthorized',
      message: 'Unauthorized',
      path: '/base/category/list',
    });
  },
  'GET /api/401': (req, res) => {
    res.status(401).send({
      timestamp: 1513932555104,
      status: 401,
      error: 'Unauthorized',
      message: 'Unauthorized',
      path: '/base/category/list',
    });
  },
};
