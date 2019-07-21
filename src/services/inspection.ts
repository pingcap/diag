import request from '@/utils/request';
import { IInstanceConfig } from '@/models/inspection';

export async function queryInstances(): Promise<any> {
  return request('/api/v1/instances');
}

export async function deleteInstance(instanceId: string): Promise<any> {
  return request.delete(`/api/v1/instances/${instanceId}`);
}

export async function queryInstanceConfig(instanceId: string): Promise<any> {
  return request(`/api/v1/instances/${instanceId}/config`);
}

export async function updateInstanceConfig(
  instanceId: string,
  config: IInstanceConfig,
): Promise<any> {
  return request.post(`/api/v1/instances/${instanceId}/config`, { data: config });
}

// /////////////

export async function queryInstanceInspections(instanceId: string, page: number = 1) {
  return request(`/api/v1/instances/${instanceId}/inspections?page=${page}`);
}

export async function queryAllInspections(page: number = 1) {
  return request(`/api/v1/inspections?page=${page}`);
}
