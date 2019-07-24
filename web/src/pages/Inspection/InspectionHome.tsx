import React from 'react';
import { connect } from 'dva';
import { Redirect } from 'umi';
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
  return <p>Loading...</p>;
}

export default connect(({ user }: ConnectState) => ({
  curUser: user.currentUser,
}))(InspectionHome);
