import React from 'react';
import { connect } from 'dva';
import { Redirect } from 'umi';
import { Spin } from 'antd';
import { ConnectState } from '@/models/connect';
import { CurrentUser } from '@/models/user';

export interface InspectionHomeProps {
  curUser: CurrentUser;
}

function InspectionHome({ curUser }: InspectionHomeProps) {
  if (curUser.role === 'admin') {
    return <Redirect to="/inspection/instances" />;
  }
  if (curUser.role === 'dba') {
    return <Redirect to="/inspection/reports" />;
  }
  return <Spin size="small" style={{ marginLeft: 8, marginRight: 8 }} />;
}

export default connect(({ user }: ConnectState) => ({
  curUser: user.currentUser,
}))(InspectionHome);
