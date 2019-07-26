import React, { useState } from 'react';
import { Button, Modal } from 'antd';
import { router } from 'umi';
import { connect } from 'dva';
import { ConnectProps, Dispatch, ConnectState } from '@/models/connect';
import { CurrentUser } from '@/models/user';
import UploadRemoteReportModal from '@/components/UploadRemoteReportModal';

const styles = require('../style.less');

interface ReportDetailProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
}

function ReportDetail({ dispatch, match, curUser }: ReportDetailProps) {
  const reportId: string | undefined = match && match.params && (match.params as any).id;

  const [uploadRemoteModalVisible, setUploadRemoteModalVisible] = useState(false);

  function deleteInspection() {
    Modal.confirm({
      title: '删除报告？',
      content: '你确定要删除这个报告吗？删除后不可恢复',
      okText: '删除',
      okButtonProps: { type: 'danger' },
      onOk() {
        dispatch({
          type: 'inspection/deleteInspection',
          payload: reportId,
        }).then(() => router.goBack());
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
              <a download href={`/api/v1/inspections/${reportId}.tar.gz`}>
                下载
              </a>
            </Button>
          </React.Fragment>
        )}
        <Button type="danger" onClick={deleteInspection}>
          删除
        </Button>
      </div>
      <div>
        <p></p>
        <p>loprem loprem</p>
        <p>loprem loprem</p>
        <p>loprem loprem</p>
        <p>loprem loprem</p>
        <p>loprem loprem</p>
        <p>loprem loprem</p>
      </div>
      <UploadRemoteReportModal
        visible={uploadRemoteModalVisible}
        onClose={() => setUploadRemoteModalVisible(false)}
        uploadUrl={`/api/v1/inspections/${reportId}`}
      />
    </div>
  );
}

export default connect(({ user }: ConnectState) => ({ curUser: user.currentUser }))(ReportDetail);
