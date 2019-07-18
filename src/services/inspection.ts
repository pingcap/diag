import request from '@/utils/request';

export async function queryInstances(): Promise<any> {
  return request('/api/v1/instances');
}

export async function deleteInstance(instanceId: string): Promise<any> {
  return request.delete(`/api/v1/instances/${instanceId}`);
}
