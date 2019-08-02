import React, { useState, useEffect } from 'react';
import { Button, Modal, Spin } from 'antd';
import { router } from 'umi';
import { connect } from 'dva';
import { ConnectProps, Dispatch, ConnectState } from '@/models/connect';
import { CurrentUser } from '@/models/user';
import UploadRemoteReportModal from '@/components/UploadRemoteReportModal';
import AutoTable from '@/components/InspectionDetail/AutoTable';
import { IInspectionDetail } from '@/models/inspection';
import { queryInspection } from '@/services/inspection';
import InspectionReport from '@/components/InspectionDetail/InspectionReport';

const styles = require('../style.less');

interface ReportDetailProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
}

function ReportDetail({ dispatch, match, curUser }: ReportDetailProps) {
  const reportId: string | undefined = match && match.params && (match.params as any).id;

  const [uploadRemoteModalVisible, setUploadRemoteModalVisible] = useState(false);
  const [inspectionDetail, setInspectionDetail] = useState<IInspectionDetail | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    async function fetchInspection() {
      if (reportId === undefined) return;
      setLoading(true);
      const res = await queryInspection(reportId);
      setLoading(false);
      if (res !== undefined) {
        setInspectionDetail(res);
      }
    }
    fetchInspection();
  }, [reportId]);

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
        {loading && <Spin size="large" style={{ marginLeft: 8, marginRight: 8 }} />}

        {inspectionDetail && <InspectionReport report={inspectionDetail.report} />}
      </div>

      <UploadRemoteReportModal
        visible={uploadRemoteModalVisible}
        onClose={() => setUploadRemoteModalVisible(false)}
        uploadUrl={`/inspections/${reportId}`}
      />
    </div>
  );
}

export default connect(({ user }: ConnectState) => ({ curUser: user.currentUser }))(ReportDetail);
