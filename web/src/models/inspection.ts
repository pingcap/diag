import { Effect } from 'dva';
import { Reducer } from 'redux';
import moment from 'moment';

import { message } from 'antd';
import {
  queryInstances,
  deleteInstance,
  queryInstanceInspections,
  deleteInspection,
  addInspection,
} from '@/services/inspection';

// /////

export interface IInstance {
  uuid: string;
  name: string;
  pd: string;
  create_time: string;
  status: string;
  message: string;
}

export interface IFormatInstance extends IInstance {
  user: string;
  key: string;
  format_create_time: string;
}

export interface IInstanceConfig {
  instance_id: string;
  collect_hardware_info: boolean; // 硬件信息
  collect_software_info: boolean; // 软件信息

  collect_log: boolean; // 应用日志信息
  collect_log_duration: number; // 应用日志信息时长

  collect_metric_duration: number; // 性能监控信息时长

  collect_demsg: boolean; // 机器 demsg 信息

  auto_sched_start: string; // 开始时间
  report_keep_duration: number; // 保存时长
}

export interface IInspection {
  uuid: string;
  instance_id: string;
  status: 'running' | 'finish';
  type: 'manual' | 'auto';
  create_time: string;
  finish_time: string;
  report_path: string;
  instance_name: string;
}

export interface IFormatInspection extends IInspection {
  user: string;
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
    user: 'default',
    key: instance.uuid,
    format_create_time: moment(instance.create_time).format('YYYY-MM-DD hh:mm'),
  };
}

function convertInstances(instances: IInstance[]): IFormatInstance[] {
  return instances.map(convertInstance);
}

function convertInspection(inspection: IInspection): IFormatInspection {
  return {
    ...inspection,
    user: 'default',
    key: inspection.uuid,
    format_create_time: moment(inspection.create_time).format('YYYY-MM-DD hh:mm'),
    format_finish_time: moment(inspection.finish_time).format('YYYY-MM-DD hh:mm'),
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
      yield put({
        type: 'saveInstances',
        payload: res,
      });
    },
    *deleteInstance({ payload }, { call, put }) {
      const instanceId = payload;
      yield call(deleteInstance, instanceId);
      yield put({
        type: 'removeInstance',
        payload,
      });
      message.success(`实例 ${instanceId} 已删除！`);
    },

    *fetchInspections({ payload }, { call, put }) {
      const { instanceId, page } = payload;
      const res: IInspectionsRes = yield call(queryInstanceInspections, instanceId, page);
      yield put({
        type: 'saveInspections',
        payload: {
          res,
          page,
        },
      });
    },
    *deleteInspection({ payload }, { call, put }) {
      const inspectionId = payload;
      yield call(deleteInspection, inspectionId);
      yield put({
        type: 'removeInspection',
        payload,
      });
      message.success(`诊断报告 ${inspectionId} 已删除！`);
      return true;
    },
    *addInspection({ _ }, { call, put }) {
      const res = yield call(addInspection);
      yield put({
        type: 'saveInspection',
        payload: res as IInspection,
      });
      message.success(`诊断 ${res.uuid} 已经进行中！`);
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
