import request from '@/utils/request';

export async function queryInstances(): Promise<any> {
  return request('/api/v1/instances');
}
