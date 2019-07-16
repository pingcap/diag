import { Effect } from 'dva';
import { Reducer } from 'redux';

import { queryInstances } from '@/services/inspection';

export interface IInstance {
  uuid: string;
  name: string;
  pd: string;
  create_time: string;
  status: string;
  message: string;
}

export interface InspectionModelState {
  instances: IInstance[];
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
      const response = yield call(queryInstances);
      yield put({
        type: 'saveInstances',
        payload: response,
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
