import React, { useEffect } from 'react';
import { connect } from 'dva';
import { Redirect } from 'umi';
import { Spin } from 'antd';
import { ConnectState, Dispatch } from '@/models/connect';
import { CurrentUser } from '@/models/user';

export interface HomePage {
  curUser: CurrentUser;
  dispatch: Dispatch;
}

function HomePage({ curUser, dispatch }: HomePage) {
  useEffect(() => {
    dispatch({
      type: 'user/fetchCurrent',
    });
  }, []);

  if (curUser.role === 'admin') {
    return <Redirect to="/instances" />;
  }
  if (curUser.role === 'dba') {
    return <Redirect to="/inspection" />;
  }
  if (curUser.username === '') {
    return <Redirect to="/user/login" />;
  }
  return <Spin size="small" style={{ marginLeft: 8, marginRight: 8 }} />;
}

export default connect(({ user }: ConnectState) => ({
  curUser: user.currentUser,
}))(HomePage);
