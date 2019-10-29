import { Effect } from 'dva';
import { Reducer } from 'redux';

import { message } from 'antd';
import {
  queryInstances,
  deleteInstance,
  queryInstanceInspections,
  deleteInspection,
  addInspection,
  queryAllInspections,
} from '@/services/inspection';
import { formatDatetime } from '@/utils/datetime-util';

// /////

export interface IInstance {
  uuid: string;
  name: string;
  user: string;
  pd: string;
  create_time: string;
  status: 'pending' | 'exception' | 'success';
  message: string;
}

export interface IFormatInstance extends IInstance {
  key: string;
  format_create_time: string;
}

export interface IInstanceConfig {
  collect_hardware_info: boolean; // 硬件信息
  collect_software_info: boolean; // 软件信息
  collect_log: boolean; // 应用日志信息
  collect_demsg: boolean; // 机器 demsg 信息

  // collect_log_duration: number; // 应用日志信息时长
  // collect_metric_duration: number; // 性能监控信息时长

  auto_sched_start: string; // 开始时间
  auto_sched_duration: number; // 统计信息时长
  auto_sched_day: string;

  manual_sched_range: [string, string]; // 手动诊断时的统计信息时间段

  report_keep_duration: number; // 保存时长
}

export interface IInspection {
  uuid: string;
  instance_id: string;
  instance_name: string;
  user: string;
  status: 'running' | 'exception' | 'success';
  message: string;
  type: 'manual' | 'auto';
  create_time: string;
  finish_time: string;

  estimated_left_sec?: number;
}

export interface IInspectionReport {
  symptoms: any[];

  basic: object;
  dbinfo?: object[];
  resource?: object[];
  alert?: object[];
  slow_log?: object[];
  hardware?: object[];
  software_version?: object[];
  software_config?: object[];
  network?: object[];
  demsg?: object[];
}

export interface IInspectionDetail extends IInspection {
  report: IInspectionReport;
  scrape_begin: string;
  scrape_end: string;
}

export interface IFormatInspection extends IInspection {
  key: string;
  format_create_time: string;
  format_finish_time: string;
}

export interface IInspectionsRes {
  total: number;
  data: IInspection[];
}

// //////

function convertInstance(instance: IInstance): IFormatInstance {
  return {
    ...instance,
    key: instance.uuid,
    format_create_time: formatDatetime(instance.create_time),
  };
}

function convertInstances(instances: IInstance[]): IFormatInstance[] {
  return instances.map(convertInstance);
}

function convertInspection(inspection: IInspection): IFormatInspection {
  return {
    ...inspection,
    key: inspection.uuid,
    format_create_time: formatDatetime(inspection.create_time),
    format_finish_time: formatDatetime(inspection.finish_time),
  };
}

function convertInspections(inspections: IInspection[]) {
  return inspections.map(convertInspection);
}

// //////

export interface InspectionModelState {
  instances: IFormatInstance[];

  inspections: IFormatInspection[];
  total_inspections: number;
  cur_inspections_page: number;
}

const initialState: InspectionModelState = {
  instances: [],

  inspections: [],
  total_inspections: 0,
  cur_inspections_page: 1,
};

export interface InspectionModelType {
  namespace: 'inspection';
  state: InspectionModelState;
  effects: {
    fetchInstances: Effect;
    deleteInstance: Effect;

    fetchInspections: Effect;
    deleteInspection: Effect;
    addInspection: Effect;
  };
  reducers: {
    saveInstances: Reducer<InspectionModelState>;
    saveInstance: Reducer<InspectionModelState>;
    removeInstance: Reducer<InspectionModelState>;

    saveInspections: Reducer<InspectionModelState>;
    saveInspection: Reducer<InspectionModelState>;
    removeInspection: Reducer<InspectionModelState>;
  };
}

// //////

const InspectionModel: InspectionModelType = {
  namespace: 'inspection',

  state: initialState,

  // effects verbs: fetch, add, delete, update
  effects: {
    *fetchInstances(_, { call, put }) {
      const res: IInstance[] = yield call(queryInstances);
      if (res !== undefined) {
        yield put({
          type: 'saveInstances',
          payload: res,
        });
      }
      return res;
    },
    *deleteInstance({ payload }, { call, put }) {
      const instanceId = payload;
      const res = yield call(deleteInstance, instanceId);
      if (res !== undefined) {
        yield put({
          type: 'removeInstance',
          payload,
        });
        message.success(`实例 ${instanceId} 已删除！`);
      }
    },

    *fetchInspections({ payload }, { call, put, select }) {
      const { instanceId } = payload;
      let { page } = payload;
      if (page === undefined) {
        page = yield select((state: any) => state.inspection.cur_inspections_page);
      }
      let res: IInspectionsRes;
      if (instanceId) {
        res = yield call(queryInstanceInspections, instanceId, page);
      } else {
        res = yield call(queryAllInspections, page);
      }
      if (res !== undefined) {
        yield put({
          type: 'saveInspections',
          payload: {
            res,
            page,
          },
        });
      }
      return res;
    },
    *deleteInspection({ payload }, { call, put }) {
      const inspectionId = payload;
      const res = yield call(deleteInspection, inspectionId);
      if (res !== undefined) {
        yield put({
          type: 'removeInspection',
          payload,
        });
        message.success(`诊断报告 ${inspectionId} 已删除！`);
      }
      return res !== undefined;
    },
    *addInspection({ payload }, { call, put }) {
      const { instanceId, config } = payload;
      const res = yield call(addInspection, instanceId, config);
      if (res !== undefined) {
        yield put({
          type: 'saveInspection',
          payload: res as IInspection,
        });
      }
      return res !== undefined;
    },
  },

  // reducers verbs: save (multiple or singal), remove, modify
  reducers: {
    saveInstances(state = initialState, { payload }) {
      return {
        ...state,

        // reset page
        cur_inspections_page: 1,
        instances: convertInstances(payload as IInstance[]),
      };
    },
    saveInstance(state = initialState, { payload }) {
      return {
        ...state,
        instances: [convertInstance(payload as IInstance)].concat(state.instances),
      };
    },
    removeInstance(state = initialState, action) {
      return {
        ...state,
        instances: state.instances.filter(item => item.uuid !== action.payload),
      };
    },

    saveInspections(state = initialState, { payload }) {
      const {
        page,
        res: { total, data },
      } = payload;
      return {
        ...state,

        total_inspections: total,
        cur_inspections_page: page,
        inspections: convertInspections(data),
      };
    },
    saveInspection(state = initialState, { payload }) {
      return {
        ...state,

        inspections: [convertInspection(payload as IInspection)]
          .concat(state.inspections)
          .slice(0, 10),
      };
    },
    removeInspection(state = initialState, { payload }) {
      const inspectionId = payload;
      return {
        ...state,

        inspections: state.inspections.filter(i => i.uuid !== inspectionId),
      };
    },
  },
};

export default InspectionModel;
