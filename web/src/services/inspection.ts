import request from '@/utils/request';
import { IInstanceConfig } from '@/models/inspection';

export async function queryInstances(): Promise<any> {
  return request('/instances');
}

export async function createInstance(config: any) {
  return request.post('/instances/text', { data: { config } });
}

export async function deleteInstance(instanceId: string): Promise<any> {
  return request.delete(`/instances/${instanceId}`);
}

export async function queryInstanceConfig(instanceId: string): Promise<any> {
  return request(`/instances/${instanceId}/config`);
}

export async function updateInstanceConfig(
  instanceId: string,
  config: IInstanceConfig,
): Promise<any> {
  return request.put(`/instances/${instanceId}/config`, { data: config });
}

// /////////////

export async function queryInstanceInspections(instanceId: string, page: number = 1) {
  return request(`/instances/${instanceId}/inspections?page=${page}`);
}

export async function addInspection(instanceId: string, config: IInstanceConfig) {
  return request.post(`/instances/${instanceId}/inspections`, { data: config });
}

export async function queryAllInspections(page: number = 1) {
  return request(`/inspections?page=${page}`);
}

export async function queryInspection(inspectionId: string) {
  return request(`/inspections/${inspectionId}`);
}

export async function deleteInspection(inspectionId: string) {
  return request.delete(`/inspections/${inspectionId}`);
}
