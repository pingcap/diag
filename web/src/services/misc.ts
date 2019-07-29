import request from '@/utils/request';

export async function queryFlamegraphs(page: number) {
  return request(`/flamegraphs?page=${page}`);
}

export async function queryFlamegraph(uuid: string) {
  return request(`/flamegraphs/${uuid}`);
}

export async function addFlamegraph(instanceId: string) {
  return request.post('/flamegraphs', { data: { instanceId } });
}

export async function deleteFlamegraph(uuid: string) {
  return request.delete(`/flamegraphs/${uuid}`);
}

export async function queryPerfProfiles(page: number) {
  return request(`/perfprofiles?page=${page}`);
}

export async function addPerfProfile(instanceId: string) {
  return request.post('/perfprofiles', { data: { instanceId } });
}

export async function deletePerfProfile(uuid: string) {
  return request.delete(`/prefprofiles/${uuid}`);
}
