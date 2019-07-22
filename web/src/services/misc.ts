import request from '@/utils/request';

export async function queryFlamegraphs(page: number) {
  return request(`/api/v1/flamegraphs?page=${page}`);
}

export async function addFlamegraph(machine: string) {
  return request.post('/api/v1/flamegraphs', { data: { machine } });
}

export async function deleteFlamegraph(uuid: string) {
  return request.delete(`/api/v1/flamegraphs/${uuid}`);
}

export async function queryPerfProfiles(page: number) {
  return request('/api/v1/perfprofiles');
}

export async function addPerfProfile(machine: string) {
  return request.post('/api/v1/perfprofiles', { data: { machine } });
}

export async function deletePerfProfile(uuid: string) {
  return request.delete(`/api/v1/prefprofiles/${uuid}`);
}
