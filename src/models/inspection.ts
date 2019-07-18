import { Effect } from 'dva';
import { Reducer } from 'redux';
import moment from 'moment';

import { queryInstances } from '@/services/inspection';

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

export interface InspectionModelState {
  instances: IFormatInstance[];
}

function convertInstances(instances: IInstance[]): IFormatInstance[] {
  return instances.map(item => ({
    ...item,
    user: 'default',
    key: item.uuid,
    format_create_time: moment(item.create_time).format('YYYY-MM-DD hh:mm'),
  }));
}

export interface InspectionModelType {
  namespace: 'inspection';
  state: InspectionModelState;
  effects: {
    fetchInstances: Effect;
  };
  reducers: {
    saveInstances: Reducer<InspectionModelState>;
  };
}

const InspectionModel: InspectionModelType = {
  namespace: 'inspection',

  state: {
    instances: [],
  },

  effects: {
    *fetchInstances(_, { call, put }) {
      const response: IInstance[] = yield call(queryInstances);
      yield put({
        type: 'saveInstances',
        payload: convertInstances(response),
      });
    },
  },

  reducers: {
    saveInstances(state, action) {
      return {
        ...state,
        instances: action.payload || [],
      };
    },
  },
};

export default InspectionModel;
