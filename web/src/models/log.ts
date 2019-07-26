import { Effect } from 'dva';
import { Reducer } from 'redux';
import { queryLogInstances, queryLogs, ILogQueryParams } from '@/services/log';
import { formatDatetime } from '@/utils/datetime-util';

// /////

export interface ILogInstance {
  uuid: string;
  name: string;
}

export interface ILog {
  time: string;
  file: string;
  content: string;
}

export interface IFormatLog extends ILog {
  key: number;
  format_time: string;
}

export interface ILogsRes {
  token: string;
  logs: ILog[];
}

function convertItem(item: ILog, lineNum: number): IFormatLog {
  return {
    ...item,
    key: lineNum,
    format_time: formatDatetime(item.time),
  };
}

// /////

export interface LogModelState {
  logInstances: ILogInstance[];
  logs: IFormatLog[];
  token: string;
}

const initialState: LogModelState = {
  logInstances: [],
  logs: [],
  token: '',
};

// /////

export interface LogModelType {
  namespace: 'log';
  state: LogModelState;
  effects: {
    fetchLogInstances: Effect;

    searchLogs: Effect;
    loadMoreLogs: Effect;
  };
  reducers: {
    saveLogInstances: Reducer<LogModelState>;

    resetLogs: Reducer<LogModelState>;
    saveLogs: Reducer<LogModelState>;
  };
}

// /////

const MiscModel: LogModelType = {
  namespace: 'log',

  state: initialState,

  effects: {
    *fetchLogInstances({ _ }, { call, put }) {
      const res: ILogInstance[] | undefined = yield call(queryLogInstances);
      if (res) {
        yield put({
          type: 'saveLogInstances',
          payload: res,
        });
      }
    },
    *searchLogs({ payload }, { call, put }) {
      yield put({ type: 'resetLogs' });

      const { logInstanceId, search, startTime, endTime } = payload;
      const parms: ILogQueryParams = {
        search,
        start_time: startTime,
        end_time: endTime,
        limit: 20,
      };
      const res: ILogsRes | undefined = yield call(queryLogs, logInstanceId, parms);
      if (res) {
        yield put({
          type: 'saveLogs',
          payload: res,
        });
      }
    },
    *loadMoreLogs({ payload }, { call, put, select }) {
      const logInstanceId = payload;
      const token = yield select((state: any) => state.log.token);
      const res: ILogsRes | undefined = yield call(queryLogs, logInstanceId, { token });
      if (res) {
        yield put({
          type: 'saveLogs',
          payload: res,
        });
      }
    },
  },
  reducers: {
    saveLogInstances(state = initialState, { payload }) {
      return {
        ...state,
        logInstances: payload,
      };
    },

    resetLogs(state = initialState, { _ }) {
      return {
        ...state,
        token: '',
        logs: [],
      };
    },
    saveLogs(state = initialState, { payload }) {
      const { token, logs } = payload as ILogsRes;
      return {
        ...state,
        token,
        logs: state.logs.concat(
          logs.map((log, index) => convertItem(log, state.logs.length + index + 1)),
        ),
      };
    },
  },
};

export default MiscModel;
