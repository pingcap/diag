import { Effect } from 'dva';
import { Reducer } from 'redux';
import {
  queryLogInstances,
  queryInstanceLogs,
  ILogQueryParams,
  queryFileLogs,
  queryLogFiles,
} from '@/services/log';
import { formatDatetime } from '@/utils/datetime-util';

// /////

export interface ILogInstance {
  uuid: string;
  instance_name: string;
}

export interface ILogFile {
  uuid: string;
  instance_name: string;
}

export interface ILog {
  time: string;
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
  logFiles: ILogFile[];

  logs: IFormatLog[];
  token: string;
}

const initialState: LogModelState = {
  logInstances: [],
  logFiles: [],

  logs: [],
  token: '',
};

// /////

export interface LogModelType {
  namespace: 'log';
  state: LogModelState;
  effects: {
    fetchLogInstances: Effect;
    fetchLogFiles: Effect;

    searchLogs: Effect;
    loadMoreLogs: Effect;
  };
  reducers: {
    saveLogInstances: Reducer<LogModelState>;
    saveLogFiles: Reducer<LogModelState>;
    saveLogFile: Reducer<LogModelState>;

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
      if (res !== undefined) {
        yield put({
          type: 'saveLogInstances',
          payload: res,
        });
      }
    },
    *fetchLogFiles({ _ }, { call, put }) {
      const res: ILogFile[] | undefined = yield call(queryLogFiles);
      if (res !== undefined) {
        yield put({
          type: 'saveLogFiles',
          payload: res,
        });
      }
    },

    *searchLogs({ payload }, { call, put }) {
      yield put({ type: 'resetLogs' });

      const { logInstanceId, logFileId, search, startTime, endTime, logLevel } = payload;
      const params: ILogQueryParams = {
        search,
        start_time: startTime,
        end_time: endTime,
        level: logLevel,
        limit: 10,
      };
      let res: ILogsRes | undefined;
      if (logInstanceId) {
        res = yield call(queryInstanceLogs, logInstanceId, params);
      } else {
        res = yield call(queryFileLogs, logFileId, params);
      }
      if (res !== undefined) {
        yield put({
          type: 'saveLogs',
          payload: res,
        });
      }
    },
    *loadMoreLogs({ payload }, { call, put, select }) {
      const { logInstanceId, logFileId } = payload;
      const token = yield select((state: any) => state.log.token);
      let res: ILogsRes | undefined;
      if (logInstanceId) {
        res = yield call(queryInstanceLogs, logInstanceId, { token });
      } else {
        res = yield call(queryFileLogs, logFileId, { token });
      }
      if (res !== undefined) {
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
    saveLogFiles(state = initialState, { payload }) {
      return {
        ...state,
        logFiles: payload,
      };
    },
    saveLogFile(state = initialState, { payload }) {
      return {
        ...state,
        logFiles: [payload as ILogFile].concat(state.logFiles),
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
