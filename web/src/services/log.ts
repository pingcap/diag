import request from '@/utils/request';

export async function queryLogInstances() {
  return request('/loginstances');
}

export async function queryLogFiles() {
  return request('/logfiles');
}

// ////////////////

export interface ILogQueryParams {
  token?: string;

  search?: string;
  limit?: number;
  start_time?: string;
  end_time?: string;
  level?: string;
}

// for admin user
export async function queryInstanceLogs(logInstanceId: string, queryParams: ILogQueryParams) {
  return request(`/loginstances/${logInstanceId}/logs`, { params: queryParams });
}

// for dba user search
export async function queryFileLogs(logFileId: string, queryParams: ILogQueryParams) {
  return request(`/logfiles/${logFileId}/logs`, { params: queryParams });
}
