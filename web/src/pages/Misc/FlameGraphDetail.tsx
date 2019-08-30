import React, { useState } from 'react';
import { Button, Modal } from 'antd';
import { router } from 'umi';
import { connect } from 'dva';
import { ConnectProps, Dispatch, ConnectState } from '@/models/connect';
import { CurrentUser } from '@/models/user';
import UploadRemoteReportModal from '@/components/UploadRemoteReportModal';
import FlameGraphTable from '@/components/FlameGraphTable';

const styles = require('../style.less');

interface ReportDetailProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
}

function FlameGraphDetail({ dispatch, match, curUser }: ReportDetailProps) {
  const reportId: string | undefined = match && match.params && (match.params as any).id;

  const [uploadRemoteModalVisible, setUploadRemoteModalVisible] = useState(false);

  function deleteFlamegraph() {
    Modal.confirm({
      title: '删除报告？',
      content: '你确定要删除这个报告吗？删除后不可恢复',
      okText: '删除',
      okButtonProps: { type: 'danger' },
      onOk() {
        dispatch({
          type: 'misc/deleteFlamegraph',
          payload: reportId,
        }).then((ret: boolean) => ret && router.goBack());
      },
    });
  }

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h3>报告：{reportId}</h3>
        <div className={styles.space}></div>
        {curUser.role === 'admin' && (
          <React.Fragment>
            <Button
              type="primary"
              style={{ marginRight: 20 }}
              onClick={() => setUploadRemoteModalVisible(true)}
            >
              上传
            </Button>
            <Button type="primary" style={{ marginRight: 20 }}>
              <a download href={`/api/v1/perfprofiles/${reportId}.tar.gz`}>
                下载
              </a>
            </Button>
          </React.Fragment>
        )}
        <Button type="danger" onClick={deleteFlamegraph}>
          删除
        </Button>
      </div>
      <FlameGraphTable reportId={reportId || ''} />
      <UploadRemoteReportModal
        visible={uploadRemoteModalVisible}
        onClose={() => setUploadRemoteModalVisible(false)}
        uploadUrl={`/perfprofiles/${reportId}`}
      />
    </div>
  );
}

export default connect(({ user }: ConnectState) => ({ curUser: user.currentUser }))(
  FlameGraphDetail,
);
