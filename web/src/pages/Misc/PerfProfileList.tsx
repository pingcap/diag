import React, { useEffect, useMemo, useState } from 'react';
import { Table, Button, Divider, Modal } from 'antd';
import { connect } from 'dva';
import { Link } from 'umi';
import { PaginationConfig } from 'antd/lib/table';
import { ConnectState, ConnectProps, Dispatch } from '@/models/connect';
import { IPerfProfileInfo, IPerfProfile } from '@/models/misc';
import AddMiscReportModal from '@/components/AddMiscReportModal';
import UploadReportModal from '@/components/UploadReportModal';

const styles = require('../style.less');

const tableColumns = (onDelete: any, onUpload: any) => [
  {
    title: 'Profile 报告 ID',
    dataIndex: 'uuid',
    key: 'uuid',
  },
  {
    title: '用户名',
    dataIndex: 'user',
    key: 'user',
  },
  {
    title: 'IP : 端口',
    dataIndex: 'machine',
    key: 'machine',
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
    render: (text: any, record: IPerfProfile) => {
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
    render: (text: any, record: IPerfProfile) => (
      <span>
        <Link to={`/misc/perfprofiles/${record.uuid}`}>查看</Link>
        <Divider type="vertical" />
        <a download href={`/api/v1/perfprofiles/${record.uuid}.tar.gz`}>
          下载
        </a>
        <Divider type="vertical" />
        <a onClick={onUpload}>上传</a>
        <Divider type="vertical" />
        <a style={{ color: 'red' }} onClick={() => onDelete(record)}>
          删除
        </a>
      </span>
    ),
  },
];

interface PerfProfileListProps extends ConnectProps {
  perfprofile: IPerfProfileInfo;
  dispatch: Dispatch;
  loading: boolean;
}

function PerfProfileList({ perfprofile, dispatch, loading }: PerfProfileListProps) {
  const [modalVisble, setModalVisible] = useState(false);

  // upload
  const [uploadModalVisible, setUploadModalVisible] = useState(false);
  const [uploadUrl, setUploadUrl] = useState('');

  const pagination: PaginationConfig = useMemo(
    () => ({
      total: perfprofile.total,
      current: perfprofile.cur_page,
    }),
    [perfprofile.total, perfprofile.cur_page],
  );

  useEffect(() => {
    fetchPerfProfiles(perfprofile.cur_page);
  }, []);

  function fetchPerfProfiles(page: number) {
    dispatch({
      type: 'misc/fetchPerfProfiles',
      payload: {
        page,
      },
    });
  }

  const columns = useMemo(() => tableColumns(deletePerfProfile, uploadPerfProfile), []);

  function deletePerfProfile(record: IPerfProfile) {
    Modal.confirm({
      title: '删除 Profile 报告？',
      content: '你确定要删除这个 Profile 报告吗？删除后不可恢复',
      okText: '删除',
      okButtonProps: { type: 'danger' },
      onOk() {
        dispatch({
          type: 'misc/deletePerfProfile',
          payload: record.uuid,
        });
      },
    });
  }

  function uploadPerfProfile(record: IPerfProfile) {
    setUploadModalVisible(true);
    setUploadUrl(`/api/v1/flamegraphs/${record.uuid}`);
  }

  function handleAddPerfProfile(machine: string): Promise<any> {
    return new Promise((resolve, reject) => {
      dispatch({
        type: 'misc/addPerfProfile',
        payload: machine,
      }).then((val: any) => resolve());
    });
  }

  function handleTableChange(curPagination: PaginationConfig) {
    fetchPerfProfiles(curPagination.current as number);
  }

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h2>Perf Profile 报告列表</h2>
        <Button type="primary" onClick={() => setModalVisible(true)}>
          + 获取
        </Button>
      </div>
      <Table
        loading={loading}
        dataSource={perfprofile.list}
        columns={columns}
        onChange={handleTableChange}
        pagination={pagination}
      />
      <AddMiscReportModal
        visible={modalVisble}
        onClose={() => setModalVisible(false)}
        onData={handleAddPerfProfile}
      />
      <UploadReportModal
        visible={uploadModalVisible}
        onClose={() => setUploadModalVisible(false)}
        uploadUrl={uploadUrl}
      />
    </div>
  );
}

export default connect(({ misc, loading }: ConnectState) => ({
  perfprofile: misc.perfprofile,
  loading: loading.effects['misc/fetchPerfProfiles'],
}))(PerfProfileList);
