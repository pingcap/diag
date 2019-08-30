import request from '@/utils/request';

export async function accountLogin(loginInfo: { username: string; password: string }) {
  const res = await request.post('/login', {
    data: loginInfo,
    errorHandler: error => {
      const { response } = error;
      console.log(response.status);
    },
  });
  return res;
}

export async function accountLogout() {
  return request.delete('/logout');
}

export async function queryCurrent(): Promise<any> {
  return request('/me');
}

export async function queryNotices(): Promise<any> {
  return request('/notices');
}
