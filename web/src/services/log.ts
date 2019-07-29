import request from '@/utils/request';

export async function queryLogInstances() {
  return request('/loginstances');
}

export interface ILogQueryParams {
  token?: string;

  search?: string;
  limit?: number;
  start_time?: string;
  end_time?: string;
  level?: string;
}

// for admin user
export async function queryLogs(logInstanceId: string, queryParams: ILogQueryParams) {
  return request(`/loginstances/${logInstanceId}/logs`, { params: queryParams });
}

// for dba user search
export async function queryUploadedLogs(logId: string, queryParams: ILogQueryParams) {
  return request(`/logs/${logId}`, { params: queryParams });
}
