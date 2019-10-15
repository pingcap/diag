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
  instance_name: string;
  user: string;
  status: 'running' | 'exception' | 'success';
  message: string;
  create_time: string;
  finish_time: string;

  items: IFlameGraphItem[];
}

export interface IFormatFlameGraph extends IFlameGraph {
  key: string;
  format_create_time: string;
  format_finish_time: string;
}

export interface IFlameGraphItem {
  address: string;
  component: string;
  flames: string[];
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
      if (res !== undefined) {
        yield put({
          type: 'saveFlamegraphs',
          payload: { page, res },
        });
      }
    },
    *addFlamegraph({ payload }, { call, put }) {
      const instanceId = payload;
      const res = yield call(addFlamegraph, instanceId);
      if (res !== undefined) {
        yield put({
          type: 'saveFlamegraph',
          payload: res,
        });
      }
      return res !== undefined;
    },
    *deleteFlamegraph({ payload }, { call, put }) {
      const uuid = payload;
      const res = yield call(deleteFlamegraph, uuid);
      if (res !== undefined) {
        yield put({
          type: 'removeFlamegraph',
          payload,
        });
        message.success(`火焰图报告 ${uuid} 已删除！`);
      }
      return res !== undefined;
    },

    *fetchPerfProfiles({ payload }, { call, put, select }) {
      let { page } = payload;
      if (page === undefined) {
        page = yield select((state: any) => state.misc.perfprofile.cur_page);
      }
      const res = yield call(queryPerfProfiles, page);
      if (res !== undefined) {
        yield put({
          type: 'savePerfProfiles',
          payload: { page, res },
        });
      }
    },
    *addPerfProfile({ payload }, { call, put }) {
      const { instanceId, node } = payload;
      const res = yield call(addPerfProfile, instanceId, node);
      if (res !== undefined) {
        yield put({
          type: 'savePerfProfile',
          payload: res,
        });
      }
      return res !== undefined;
    },
    *deletePerfProfile({ payload }, { call, put }) {
      const uuid = payload;
      const res = yield call(deletePerfProfile, uuid);
      if (res !== undefined) {
        yield put({
          type: 'removePerfProfile',
          payload,
        });
        message.success(`Perf Profile 报告 ${uuid} 已删除！`);
      }
      return res !== undefined;
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
