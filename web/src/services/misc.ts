import request from '@/utils/request';

export async function queryFlamegraphs(page: number) {
  // return request(`/flamegraphs?page=${page}`);
  return request(`/perfprofiles?page=${page}`);
}

export async function queryFlamegraph(uuid: string) {
  // return request(`/flamegraphs/${uuid}`);
  return request(`/perfprofiles/${uuid}`);
}

export async function addFlamegraph(instanceId: string) {
  // return request.post('/flamegraphs', { data: { instanceId } });
  return request.post(`/instances/${instanceId}/perfprofiles`);
}

export async function deleteFlamegraph(uuid: string) {
  // return request.delete(`/flamegraphs/${uuid}`);
  return request.delete(`/perfprofiles/${uuid}`);
}

// /////////////////////

export async function queryPerfProfiles(page: number) {
  return request(`/perfprofiles?page=${page}`);
}

export async function addPerfProfile(instanceId: string, node: string) {
  if (node === 'all') {
    return request.post(`/instances/${instanceId}/perfprofiles`);
  }
  return request.post(`/instances/${instanceId}/perfprofiles?node=${node}`);
}

export async function deletePerfProfile(uuid: string) {
  return request.delete(`/perfprofiles/${uuid}`);
}

// /////////////////////

export async function queryInstanceComponents(instanceId: string) {
  return request(`/instances/${instanceId}/components`);
}
