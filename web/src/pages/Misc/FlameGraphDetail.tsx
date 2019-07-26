import React, { useState, useEffect } from 'react';
import { Button, Modal, Spin } from 'antd';
import { router } from 'umi';
import { connect } from 'dva';
import { ConnectProps, Dispatch, ConnectState } from '@/models/connect';
import { CurrentUser } from '@/models/user';
import UploadRemoteReportModal from '@/components/UploadRemoteReportModal';
import { IFlameGraphDetail } from '@/models/misc';
import { queryFlamegraph } from '@/services/misc';

const styles = require('../style.less');

interface ReportDetailProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
}

function FlameGraph({ detail }: { detail: IFlameGraphDetail }) {
  return (
    <div>
      <p>click to view in a new tab</p>
      <a href={detail.svg_url} target="_blank" rel="noopener noreferrer">
        <img alt="flamegraph" src={detail.image_url} style={{ maxWidth: '100%' }} />
      </a>
    </div>
  );
}

function FlameGraphDetail({ dispatch, match, curUser }: ReportDetailProps) {
  const reportId: string | undefined = match && match.params && (match.params as any).id;

  const [uploadRemoteModalVisible, setUploadRemoteModalVisible] = useState(false);
  const [detail, setDetail] = useState<IFlameGraphDetail | null>(null);

  useEffect(() => {
    async function fetchDetail() {
      if (reportId) {
        const res: IFlameGraphDetail | undefined = await queryFlamegraph(reportId);
        if (res) {
          setDetail(res);
        }
      }
    }
    fetchDetail();
  }, []);

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
              <a download href={`/api/v1/flamegraphs/${reportId}.tar.gz`}>
                下载
              </a>
            </Button>
          </React.Fragment>
        )}
        <Button type="danger" onClick={deleteFlamegraph}>
          删除
        </Button>
      </div>
      <section className={styles.report_detail_body}>
        {detail ? (
          <FlameGraph detail={detail} />
        ) : (
          <Spin size="small" style={{ marginLeft: 8, marginRight: 8 }} />
        )}
      </section>
      <UploadRemoteReportModal
        visible={uploadRemoteModalVisible}
        onClose={() => setUploadRemoteModalVisible(false)}
        uploadUrl={`/api/v1/flamegraphs/${reportId}`}
      />
    </div>
  );
}

export default connect(({ user }: ConnectState) => ({ curUser: user.currentUser }))(
  FlameGraphDetail,
);
