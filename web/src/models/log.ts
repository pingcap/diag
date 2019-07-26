import { Effect } from 'dva';
import { Reducer } from 'redux';
import { queryLogInstances, queryLogs, ILogQueryParams, queryUploadedLogs } from '@/services/log';
import { formatDatetime } from '@/utils/datetime-util';

// /////

export interface ILogInstance {
  uuid: string;
  name: string;
}

export interface ILog {
  time: string;
  instance_name: string;
  level: string;
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

      const { logInstanceId, logId, search, startTime, endTime, logLevel } = payload;
      const parms: ILogQueryParams = {
        search,
        start_time: startTime,
        end_time: endTime,
        level: logLevel,
        limit: 10,
      };
      let res: ILogsRes | undefined;
      if (logInstanceId) {
        res = yield call(queryLogs, logInstanceId, parms);
      } else {
        res = yield call(queryUploadedLogs, logId, parms);
      }
      if (res) {
        yield put({
          type: 'saveLogs',
          payload: res,
        });
      }
    },
    *loadMoreLogs({ payload }, { call, put, select }) {
      const { logInstanceId, logId } = payload;
      const token = yield select((state: any) => state.log.token);
      let res: ILogsRes | undefined;
      if (logInstanceId) {
        res = yield call(queryLogs, logInstanceId, { token });
      } else {
        res = yield call(queryUploadedLogs, logId, { token });
      }
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
