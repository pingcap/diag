import request from '@/utils/request';

export async function queryEmphasisList(page: number) {
  return request(`/emphasis?page=${page}`);
}

export async function queryInstanceEmphasisList(instanceId: string, page: number) {
  return request(`/instances/${instanceId}/emphasis?page=${page}`);
}

export async function queryEmphasisDetail(uuid: string) {
  return request(`/emphasis/${uuid}`);
}

export async function addEmphasis(instanceId: string) {
  return request.post(`/instances/${instanceId}/emphasis`);
}

export async function deleteEmphasis(uuid: string) {
  return request.delete(`/emphasis/${uuid}`);
}

// 本地上传 post /emphasis
// 远程上传 put /emphasis/:id
