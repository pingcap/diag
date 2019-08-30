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

function PerfProfileDetail({ dispatch, match, curUser }: ReportDetailProps) {
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
          type: 'misc/deletePerfProfile',
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
          <Button
            type="primary"
            style={{ marginRight: 20 }}
            onClick={() => setUploadRemoteModalVisible(true)}
          >
            上传
          </Button>
        )}
        <Button type="primary" style={{ marginRight: 20 }}>
          <a download href={`/api/v1/perfprofiles/${reportId}.tar.gz`}>
            下载
          </a>
        </Button>
        <Button type="danger" onClick={deleteFlamegraph}>
          删除
        </Button>
      </div>
      <section className={styles.report_detail_body}>
        <p>如需更进一步的分析请点击 &quot;下载&quot; 按钮将报告下载到本地分析。</p>
      </section>
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
  PerfProfileDetail,
);
