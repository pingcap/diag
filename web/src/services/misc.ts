import request from '@/utils/request';

export async function queryFlamegraphs(page: number) {
  return request(`/api/v1/flamegraphs?page=${page}`);
}

export async function queryFlamegraph(uuid: string) {
  return request(`/api/v1/flamegraphs/${uuid}`);
}

export async function addFlamegraph(instanceId: string) {
  return request.post('/api/v1/flamegraphs', { data: { instanceId } });
}

export async function deleteFlamegraph(uuid: string) {
  return request.delete(`/api/v1/flamegraphs/${uuid}`);
}

export async function queryPerfProfiles(page: number) {
  return request(`/api/v1/perfprofiles?page=${page}`);
}

export async function addPerfProfile(instanceId: string) {
  return request.post('/api/v1/perfprofiles', { data: { instanceId } });
}

export async function deletePerfProfile(uuid: string) {
  return request.delete(`/api/v1/prefprofiles/${uuid}`);
}
