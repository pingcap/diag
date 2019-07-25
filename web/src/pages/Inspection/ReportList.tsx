import React, { useEffect, useMemo, useState } from 'react';
import { Table, Button, Divider, Modal } from 'antd';
import { connect } from 'dva';
import { Link } from 'umi';
import { PaginationConfig } from 'antd/lib/table';
import { ConnectState, ConnectProps, InspectionModelState, Dispatch } from '@/models/connect';
import { IFormatInspection } from '@/models/inspection';
import UploadReportModal from '@/components/UploadReportModal';
import { CurrentUser } from '@/models/user';

const styles = require('../style.less');

const tableColumns = (curUser: CurrentUser, onDelete: any, onUpload: any) => [
  {
    title: '诊断报告 ID',
    dataIndex: 'uuid',
    key: 'uuid',
  },
  {
    title: '用户名',
    dataIndex: 'user',
    key: 'user',
  },
  {
    title: '实例名称',
    dataIndex: 'instance_name',
    key: 'instance_name',
  },
  {
    title: '诊断方式',
    dataIndex: 'type',
    key: 'type',
  },
  {
    title: '开始时间',
    dataIndex: 'format_create_time',
    key: 'format_create_time',
  },
  {
    title: '完成时间',
    dataIndex: 'format_finish_time',
    key: 'format_finish_time',
    render: (text: any, record: IFormatInspection) => {
      if (record.status === 'running') {
        return <span>running...</span>;
      }
      return <span>{text}</span>;
    },
  },
  {
    title: '报告保存地址',
    dataIndex: 'report_path',
    key: 'report_path',
  },
  {
    title: '操作',
    key: 'action',
    render: (text: any, record: IFormatInspection) => (
      <span>
        {curUser.role === 'dba' && <Link to={`/inspection/reports/${record.uuid}`}>详情</Link>}
        {curUser.role === 'admin' && (
          <React.Fragment>
            {record.status === 'running' ? (
              <span>详情</span>
            ) : (
              <Link to={`/inspection/reports/${record.uuid}`}>详情</Link>
            )}
            <Divider type="vertical" />
            {record.status === 'running' ? (
              <span>下载</span>
            ) : (
              <a download href={`/api/v1/inspections/${record.uuid}.tar.gz`}>
                下载
              </a>
            )}
            {curUser.ka && (
              <React.Fragment>
                <Divider type="vertical" />
                {record.status === 'running' ? <span>上传</span> : <a onClick={onUpload}>上传</a>}
              </React.Fragment>
            )}
          </React.Fragment>
        )}
        <Divider type="vertical" />
        <a style={{ color: 'red' }} onClick={() => onDelete(record)}>
          删除
        </a>
      </span>
    ),
  },
];

interface ReportListProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
  inspection: InspectionModelState;
  loading: boolean;
  inspecting: boolean;
}

function ReportList({
  dispatch,
  curUser,
  inspection,
  match,
  loading,
  inspecting,
}: ReportListProps) {
  const [modalVisible, setModalVisible] = useState(false);
  const [uploadUrl, setUploadUrl] = useState('');

  const pagination: PaginationConfig = useMemo(
    () => ({
      total: inspection.total_inspections,
      current: inspection.cur_inspections_page,
    }),
    [inspection.cur_inspections_page, inspection.total_inspections],
  );

  useEffect(() => {
    fetchInspections(inspection.cur_inspections_page);
  }, []);

  function fetchInspections(page: number) {
    const instanceId: string | undefined = match && match.params && (match.params as any).id;
    dispatch({
      type: 'inspection/fetchInspections',
      payload: {
        page,
        instanceId,
      },
    });
  }

  const columns = useMemo(() => tableColumns(curUser, deleteInspection, uploadInspection), [
    curUser,
  ]);

  function deleteInspection(record: IFormatInspection) {
    Modal.confirm({
      title: '删除报告？',
      content: '你确定要删除这个报告吗？删除后不可恢复',
      okText: '删除',
      okButtonProps: { type: 'danger' },
      onOk() {
        dispatch({
          type: 'inspection/deleteInspection',
          payload: record.uuid,
        });
      },
    });
  }

  function uploadInspection(record: IFormatInspection) {
    setModalVisible(true);
    setUploadUrl(`/api/v1/inspections/${record.uuid}`);
  }

  function manuallyInspect() {
    Modal.confirm({
      title: '手动诊断？',
      content: '你确定要发起一次手动诊断吗？',
      okText: '诊断',
      onOk() {
        dispatch({
          type: 'inspection/addInspection',
        });
      },
    });
  }

  function handleTableChange(curPagination: PaginationConfig) {
    fetchInspections(curPagination.current as number);
  }

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h2>诊断报告列表</h2>
        <Button type="primary" onClick={manuallyInspect} loading={inspecting}>
          手动一键诊断
        </Button>
      </div>
      <Table
        loading={loading}
        dataSource={inspection.inspections}
        columns={columns}
        onChange={handleTableChange}
        pagination={pagination}
      />
      <UploadReportModal
        visible={modalVisible}
        onClose={() => setModalVisible(false)}
        uploadUrl={uploadUrl}
      />
    </div>
  );
}

export default connect(({ user, inspection, loading }: ConnectState) => ({
  curUser: user.currentUser,
  inspection,
  loading: loading.effects['inspection/fetchInspections'],
  inspecting: loading.effects['inspection/addInspection'],
}))(ReportList);
