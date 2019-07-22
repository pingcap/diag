import React from 'react';
import { Button, Modal } from 'antd';
import { router } from 'umi';
import { connect } from 'dva';
import { ConnectProps, Dispatch } from '@/models/connect';

const styles = require('../style.less');

interface ReportDetailProps extends ConnectProps {
  dispatch: Dispatch;
}

function ReportDetail({ dispatch, match }: ReportDetailProps) {
  const reportId: string | undefined = match && match.params && (match.params as any).id;

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
        <Button type="primary" style={{ marginRight: 20 }}>
          <a download href={`/api/v1/inspections/${reportId}.tar.gz`}>
            拷贝
          </a>
        </Button>
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
    </div>
  );
}

export default connect()(ReportDetail);
