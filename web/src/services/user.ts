import request from '@/utils/request';

export async function accountLogin(loginInfo: { username: string; password: string }) {
  return request.post('/api/v1/login', { data: loginInfo });
}

export async function accountLogout() {
  return request.post('/api/v1/logout');
}

export async function queryCurrent(): Promise<any> {
  return request('/api/v1/me');
}

export async function queryNotices(): Promise<any> {
  return request('/api/v1/notices');
}
