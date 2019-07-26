import { Effect } from 'dva';
import { Reducer } from 'redux';
import { message } from 'antd';
import {
  queryFlamegraphs,
  deleteFlamegraph,
  addFlamegraph,
  queryPerfProfiles,
  addPerfProfile,
  deletePerfProfile,
} from '@/services/misc';
import { formatDatetime } from '@/utils/datetime-util';

// /////

export interface IFlameGraph {
  uuid: string;
  machine: string;
  user: string;
  status: 'running' | 'finish';
  create_time: string;
  finish_time: string;
}

export interface IFormatFlameGraph extends IFlameGraph {
  key: string;
  format_create_time: string;
  format_finish_time: string;
}

export type IPerfProfile = IFlameGraph;
export type IFormatPerfProfile = IFormatFlameGraph;

function convertItem(item: IFlameGraph): IFormatFlameGraph {
  return {
    ...item,
    key: item.uuid,
    format_create_time: formatDatetime(item.create_time),
    format_finish_time: formatDatetime(item.finish_time),
  };
}

// /////

export interface IFlameGraphInfo {
  list: IFormatFlameGraph[];
  total: number;
  cur_page: number;
}

export type IPerfProfileInfo = IFlameGraphInfo;

export interface MiscModelState {
  flamegraph: IFlameGraphInfo;
  perfprofile: IPerfProfileInfo;
}

const initialState: MiscModelState = {
  flamegraph: {
    list: [],
    total: 0,
    cur_page: 1,
  },
  perfprofile: {
    list: [],
    total: 0,
    cur_page: 1,
  },
};

// /////

export interface MiscModelType {
  namespace: 'misc';
  state: MiscModelState;
  effects: {
    fetchFlamegraphs: Effect;
    addFlamegraph: Effect;
    deleteFlamegraph: Effect;

    fetchPerfProfiles: Effect;
    addPerfProfile: Effect;
    deletePerfProfile: Effect;
  };
  reducers: {
    saveFlamegraphs: Reducer<MiscModelState>;
    saveFlamegraph: Reducer<MiscModelState>;
    removeFlamegraph: Reducer<MiscModelState>;

    savePerfProfiles: Reducer<MiscModelState>;
    savePerfProfile: Reducer<MiscModelState>;
    removePerfProfile: Reducer<MiscModelState>;
  };
}

// /////

const MiscModel: MiscModelType = {
  namespace: 'misc',

  state: initialState,

  effects: {
    *fetchFlamegraphs({ payload }, { call, put }) {
      const { page } = payload;
      const res = yield call(queryFlamegraphs, page);
      yield put({
        type: 'saveFlamegraphs',
        payload: { page, res },
      });
    },
    *addFlamegraph({ payload }, { call, put }) {
      const machine = payload;
      const res = yield call(addFlamegraph, machine);
      yield put({
        type: 'saveFlamegraph',
        payload: res,
      });
      return true;
    },
    *deleteFlamegraph({ payload }, { call, put }) {
      const uuid = payload;
      yield call(deleteFlamegraph, uuid);
      yield put({
        type: 'removeFlamegraph',
        payload,
      });
      message.success(`火焰图报告 ${uuid} 已删除！`);
      return true;
    },

    *fetchPerfProfiles({ payload }, { call, put }) {
      const { page } = payload;
      const res = yield call(queryPerfProfiles, page);
      yield put({
        type: 'savePerfProfiles',
        payload: { page, res },
      });
    },
    *addPerfProfile({ payload }, { call, put }) {
      const machine = payload;
      const res = yield call(addPerfProfile, machine);
      yield put({
        type: 'savePerfProfile',
        payload: res,
      });
      return true;
    },
    *deletePerfProfile({ payload }, { call, put }) {
      const uuid = payload;
      yield call(deletePerfProfile, uuid);
      yield put({
        type: 'removePerfProfile',
        payload,
      });
      message.success(`Perf Profile 报告 ${uuid} 已删除！`);
      return true;
    },
  },
  reducers: {
    saveFlamegraphs(state = initialState, { payload }) {
      const {
        page,
        res: { total, data },
      } = payload;
      return {
        ...state,
        flamegraph: {
          ...state.flamegraph,
          total,
          cur_page: page,
          list: (data as IFlameGraph[]).map(convertItem),
        },
      };
    },
    saveFlamegraph(state = initialState, { payload }) {
      return {
        ...state,
        flamegraph: {
          ...state.flamegraph,
          list: [convertItem(payload as IFlameGraph)].concat(state.flamegraph.list).slice(0, 9),
        },
      };
    },
    removeFlamegraph(state = initialState, { payload }) {
      const uuid = payload;
      return {
        ...state,
        flamegraph: {
          ...state.flamegraph,
          list: state.flamegraph.list.filter(item => item.uuid !== uuid),
        },
      };
    },

    savePerfProfiles(state = initialState, { payload }) {
      const {
        page,
        res: { total, data },
      } = payload;
      return {
        ...state,
        perfprofile: {
          ...state.perfprofile,
          total,
          cur_page: page,
          list: (data as IPerfProfile[]).map(convertItem),
        },
      };
    },
    savePerfProfile(state = initialState, { payload }) {
      return {
        ...state,
        perfprofile: {
          ...state.perfprofile,
          list: [convertItem(payload as IPerfProfile)].concat(state.perfprofile.list).slice(0, 9),
        },
      };
    },
    removePerfProfile(state = initialState, { payload }) {
      const uuid = payload;
      return {
        ...state,
        perfprofile: {
          ...state.perfprofile,
          list: state.perfprofile.list.filter(item => item.uuid !== uuid),
        },
      };
    },
  },
};

export default MiscModel;
