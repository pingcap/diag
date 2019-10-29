import { Effect } from 'dva';
import { Reducer } from 'redux';
import { message } from 'antd';
import {
  queryEmphasisList,
  addEmphasis,
  deleteEmphasis,
  queryInstanceEmphasisList,
} from '@/services/emphasis';

// /////

export interface IEmphasis {
  uuid: string;
  user: string;
  instance_name: string;

  start_time: string;
  finish_time: string;

  create_time: string;

  status: 'running' | 'exception' | 'success';
  message: string;
}

export interface IEmphasisDetail extends IEmphasis {
  scrape_begin: string;
  scrape_end: string;
}

// /////

export interface IEmphasisInfo {
  list: IEmphasis[];
  total: number;
  cur_page: number;
}

export interface IEmphasisRes {
  total: number;
  data: IEmphasis[];
}

export interface EmphasisModelState {
  emphasis: IEmphasisInfo;
}

const initialState: EmphasisModelState = {
  emphasis: {
    list: [],
    total: 0,
    cur_page: 1,
  },
};

// /////

export interface EmphasisModelType {
  namespace: 'emphasis';
  state: EmphasisModelState;
  effects: {
    fetchEmphasisList: Effect;
    addEmphasis: Effect;
    deleteEmphasis: Effect;
  };
  reducers: {
    saveEmphasisList: Reducer<EmphasisModelState>;
    saveEmphasis: Reducer<EmphasisModelState>;
    removeEmphasis: Reducer<EmphasisModelState>;
  };
}

// /////

const EmphasisModel: EmphasisModelType = {
  namespace: 'emphasis',

  state: initialState,

  effects: {
    *fetchEmphasisList({ payload }, { call, put, select }) {
      const { instanceId } = payload;
      let { page } = payload;
      if (page === undefined) {
        page = yield select((state: any) => state.emphasis.emphasis.cur_page);
      }
      let res: IEmphasisRes;
      if (instanceId) {
        res = yield call(queryInstanceEmphasisList, instanceId, page);
      } else {
        res = yield call(queryEmphasisList, page);
      }
      if (res !== undefined) {
        yield put({
          type: 'saveEmphasisList',
          payload: { page, res },
        });
      }
      return res;
    },
    *addEmphasis({ payload }, { call, put }) {
      const instanceId = payload;
      const res = yield call(addEmphasis, instanceId);
      if (res !== undefined) {
        yield put({
          type: 'saveEmphasis',
          payload: res,
        });
      }
      return res !== undefined;
    },
    *deleteEmphasis({ payload }, { call, put }) {
      const uuid = payload;
      const res = yield call(deleteEmphasis, uuid);
      if (res !== undefined) {
        yield put({
          type: 'removeEmphasis',
          payload,
        });
        message.success(`重点问题报告 ${uuid} 已删除！`);
      }
      return res !== undefined;
    },
  },
  reducers: {
    saveEmphasisList(state = initialState, { payload }) {
      const {
        page,
        res: { total, data },
      } = payload;
      return {
        ...state,
        emphasis: {
          ...state.emphasis,
          total,
          cur_page: page,
          list: data as IEmphasis[],
        },
      };
    },
    saveEmphasis(state = initialState, { payload }) {
      return {
        ...state,
        emphasis: {
          ...state.emphasis,
          list: [payload as IEmphasis].concat(state.emphasis.list).slice(0, 9),
        },
      };
    },
    removeEmphasis(state = initialState, { payload }) {
      const uuid = payload;
      return {
        ...state,
        emphasis: {
          ...state.emphasis,
          list: state.emphasis.list.filter(item => item.uuid !== uuid),
        },
      };
    },
  },
};

export default EmphasisModel;
