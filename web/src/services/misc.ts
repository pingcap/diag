import request from '@/utils/request';

export async function queryFlamegraphs(page: number) {
  return request(`/api/v1/flamegraphs?page=${page}`);
}

export async function addFlamegraph() {
  return request.post('/api/v1/flamegraphs');
}

export async function deleteFlamegraph(uuid: string) {
  return request.delete(`/api/v1/flamegraphs/${uuid}`);
}

export async function queryPerfProfiles(page: number) {
  return request('/api/v1/perfprofiles');
}

export async function addPerfProfile() {
  return request.post('/api/v1/perfprofiles');
}

export async function deletePerfProfile(uuid: string) {
  return request.delete(`/api/v1/prefprofiles/${uuid}`);
}
