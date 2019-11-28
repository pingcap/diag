import React, { useState, useEffect } from 'react';
import { Button, Modal, Spin } from 'antd';
import { router } from 'umi';
import { connect } from 'dva';
import UploadRemoteReportModal from '@/components/UploadRemoteReportModal';
import { ConnectProps, Dispatch, ConnectState } from '@/models/connect';
import { CurrentUser } from '@/models/user';
import { IEmphasisDetail } from '@/models/emphasis';
import { queryEmphasisDetail } from '@/services/emphasis';
import EmphasisReport from '@/components/ReportDetail/EmphasisReport';
import { formatDatetime } from '@/utils/datetime-util';

const styles = require('../style.less');

interface EmphasisDetailProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
}

function EmphasisDetail({ dispatch, match, curUser }: EmphasisDetailProps) {
  const reportId: string | undefined = match && match.params && (match.params as any).id;

  const [uploadRemoteModalVisible, setUploadRemoteModalVisible] = useState(false);
  const [emphasisDetail, setEmphasisDetail] = useState<IEmphasisDetail | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    async function fetchEmphasis() {
      if (reportId === undefined) return;
      setLoading(true);
      const res = await queryEmphasisDetail(reportId);
      setLoading(false);
      if (res !== undefined) {
        setEmphasisDetail(res);
      }
    }
    fetchEmphasis();
  }, [reportId]);

  function deleteEmphasis() {
    Modal.confirm({
      title: '删除报告？',
      content: '你确定要删除这个报告吗？删除后不可恢复',
      okText: '删除',
      okButtonProps: { type: 'danger' },
      onOk() {
        dispatch({
          type: 'emphasis/deleteEmphasis',
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
              <a download href={`/api/v1/emphasis/zip/${reportId}.tar.gz`}>
                下载
              </a>
            </Button>
          </React.Fragment>
        )}
        <Button type="danger" onClick={deleteEmphasis}>
          删除
        </Button>
      </div>

      <div>
        <h2>集群：{emphasisDetail && emphasisDetail.instance_name}</h2>
        <h2>
          排查时间段：
          {emphasisDetail &&
            `${formatDatetime(emphasisDetail.investgating_start)} ~
          ${formatDatetime(emphasisDetail.investgating_end)}`}
        </h2>
        <h2>排查问题：{emphasisDetail && emphasisDetail.investgating_problem}</h2>
      </div>

      <div>
        {loading && <Spin size="large" style={{ marginLeft: 8, marginRight: 8 }} />}

        {emphasisDetail && <EmphasisReport emphasis={emphasisDetail} />}
      </div>

      <UploadRemoteReportModal
        visible={uploadRemoteModalVisible}
        onClose={() => setUploadRemoteModalVisible(false)}
        uploadUrl={`/emphasis/${reportId}`}
      />
    </div>
  );
}

export default connect(({ user }: ConnectState) => ({ curUser: user.currentUser }))(EmphasisDetail);
