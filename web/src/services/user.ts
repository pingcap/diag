import request from '@/utils/request';

export async function accountLogin(loginInfo: { username: string; password: string }) {
  return request.post('/login', { data: loginInfo });
}

export async function accountLogout() {
  return request.post('/logout');
}

export async function queryCurrent(): Promise<any> {
  return request('/me');
}

export async function queryNotices(): Promise<any> {
  return request('/notices');
}
