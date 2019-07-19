import { Effect } from 'dva';
import { Reducer } from 'redux';
import moment from 'moment';

import { message } from 'antd';
import { queryInstances, deleteInstance } from '@/services/inspection';

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
  instance_uuid: string;
  collect_hardware_info: boolean; // 硬件信息
  collect_software_info: boolean; // 软件信息

  collect_log: boolean; // 应用日志信息
  collect_log_duration: number; // 应用日志信息时长

  collect_metric_duration: number; // 性能监控信息时长

  collect_demsg: boolean; // 机器 demsg 信息

  auto_sched_start: string; // 开始时间
  report_keep_duration: number; // 保存时长
}

// //////

export interface InspectionModelState {
  instances: IFormatInstance[];
}

const initialState: InspectionModelState = {
  instances: [],
};

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

export interface InspectionModelType {
  namespace: 'inspection';
  state: InspectionModelState;
  effects: {
    fetchInstances: Effect;
    deleteInstance: Effect;
  };
  reducers: {
    saveInstances: Reducer<InspectionModelState>;
    saveInstance: Reducer<InspectionModelState>;
    removeInstance: Reducer<InspectionModelState>;
  };
}

const InspectionModel: InspectionModelType = {
  namespace: 'inspection',

  state: initialState,

  // effects verbs: fetch, add, delete, update
  effects: {
    *fetchInstances(_, { call, put }) {
      const response: IInstance[] = yield call(queryInstances);
      yield put({
        type: 'saveInstances',
        payload: convertInstances(response),
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
  },

  // reducers verbs: save (multiple or singal), remove, modify
  reducers: {
    saveInstances(state, action) {
      return {
        ...state,
        instances: action.payload || [],
      };
    },
    saveInstance(state = initialState, action) {
      const instance = action.payload as IInstance;
      return {
        ...state,
        instances: state.instances.concat(convertInstance(instance)),
      };
    },
    removeInstance(state = initialState, action) {
      return {
        ...state,
        instances: state.instances.filter(item => item.uuid !== action.payload),
      };
    },
  },
};

export default InspectionModel;
