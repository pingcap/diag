import request from '@/utils/request';

export async function queryLogInstances() {
  return request('/api/v1/loginstances');
}

export interface ILogQueryParams {
  token?: string;

  search?: string;
  limit?: number;
  start_time?: string;
  end_time?: string;
  level?: string;
}

export async function queryLogs(logInstanceId: string, queryParams: ILogQueryParams) {
  return request(`/api/v1/loginstances/${logInstanceId}/logs`, { params: queryParams });
}
