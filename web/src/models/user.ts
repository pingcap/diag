import { Effect } from 'dva';
import { Reducer } from 'redux';

import { routerRedux } from 'dva/router';
import { queryCurrent, accountLogin, accountLogout } from '@/services/user';

export interface CurrentUser {
  username?: string;
  role?: 'admin' | 'dba';
  ka?: boolean;
}

export interface UserModelState {
  currentUser: CurrentUser;
}

export interface UserModelType {
  namespace: 'user';
  state: UserModelState;
  effects: {
    login: Effect;
    logout: Effect;
    fetchCurrent: Effect;
  };
  reducers: {
    saveCurrentUser: Reducer<UserModelState>;
  };
}

const UserModel: UserModelType = {
  namespace: 'user',

  state: {
    currentUser: {},
  },

  effects: {
    *login({ payload }, { call, put }) {
      const loginInfo = payload;
      const res = yield call(accountLogin, loginInfo);
      if (res !== undefined) {
        yield put({
          type: 'saveCurrentUser',
          payload: res,
        });
        yield put(
          routerRedux.replace({
            pathname: '/',
          }),
        );
      }
      return res;
    },
    *logout(_, { call, put }) {
      yield call(accountLogout);
      yield put(
        routerRedux.replace({
          pathname: '/user/login',
        }),
      );
      yield put({
        type: 'saveCurrentUser',
        res: {},
      });
    },
    *fetchCurrent(_, { call, put, select }) {
      const user = yield select((state: any) => state.user.currentUser);
      if (user.username) {
        return;
      }

      const res = yield call(queryCurrent);
      if (res !== undefined) {
        yield put({
          type: 'saveCurrentUser',
          payload: res,
        });
      }
    },
  },

  reducers: {
    saveCurrentUser(state, action) {
      return {
        ...state,
        currentUser: action.payload || {},
      };
    },
  },
};

export default UserModel;
