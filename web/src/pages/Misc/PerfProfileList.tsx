import React, { useMemo, useState } from 'react';
import { Table, Button, Divider, Modal, Tooltip, Icon } from 'antd';
import { connect } from 'dva';
import { Link } from 'umi';
import { PaginationConfig } from 'antd/lib/table';
import { ConnectState, ConnectProps, Dispatch } from '@/models/connect';
import { IPerfProfileInfo, IPerfProfile } from '@/models/misc';
import AddMiscReportModal from '@/components/AddMiscReportModal';
import UploadRemoteReportModal from '@/components/UploadRemoteReportModal';
import { CurrentUser } from '@/models/user';
import UploadLocalReportModal from '@/components/UploadLocalReportModal';
import { IFormatInstance } from '@/models/inspection';
import { useIntervalRun } from '@/custom-hooks/use-interval-run';

const styles = require('../style.less');

const tableColumns = (curUser: CurrentUser, onDelete: any, onUpload: any) => [
  {
    title: '报告 ID',
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
    title: '节点',
    dataIndex: 'no-need',
    key: 'node',
    render: (text: any, record: IPerfProfile) => {
      let content = '';
      if (record.items === null || record.items.length === 0) {
        content = '获取中...';
      } else if (record.items.length > 1) {
        content = 'all';
      } else {
        // record.items.length === 1
        content = `${record.items[0].component}-${record.items[0].address}`;
      }
      return <span>{content}</span>;
    },
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
      if (record.status === 'exception') {
        return (
          <div className={styles.instance_status}>
            <span style={{ color: 'red' }}>exception</span>
            <Tooltip title={record.message}>
              <Icon type="question-circle" />
            </Tooltip>
          </div>
        );
      }
      if (record.status === 'running') {
        return <span>running</span>;
      }
      return <span>{text}</span>;
    },
  },
  {
    title: '操作',
    key: 'action',
    render: (text: any, record: IPerfProfile) => (
      <span>
        {record.status === 'success' ? (
          <Link to={`/misc/perfprofiles/${record.uuid}`}>详情</Link>
        ) : (
          <span>详情</span>
        )}
        <Divider type="vertical" />
        {record.status === 'success' ? (
          <a download href={`/api/v1/perfprofiles/${record.uuid}.tar.gz`}>
            下载
          </a>
        ) : (
          <span>下载</span>
        )}
        {curUser.role === 'admin' && curUser.ka && (
          <React.Fragment>
            <Divider type="vertical" />
            {record.status === 'success' ? (
              <a onClick={() => onUpload(record)}>上传</a>
            ) : (
              <span>上传</span>
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

interface PerfProfileListProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
  perfprofile: IPerfProfileInfo;
  instances: IFormatInstance[];
  loading: boolean;
}

function PerfProfileList({
  dispatch,
  curUser,
  perfprofile,
  instances,
  loading,
}: PerfProfileListProps) {
  const [addReportModalVisble, setAddReportModalVisible] = useState(false);

  const [uploadRemoteModalVisible, setUploadRemoteModalVisible] = useState(false);
  const [uploadRemoteUrl, setUploadRemoteUrl] = useState('');

  const [uploadLocalModalVisible, setUploadLocalModalVisible] = useState(false);

  const pagination: PaginationConfig = useMemo(
    () => ({
      total: perfprofile.total,
      current: perfprofile.cur_page,
    }),
    [perfprofile.total, perfprofile.cur_page],
  );

  const columns = useMemo(() => tableColumns(curUser, deletePerfProfile, uploadPerfProfile), [
    curUser,
  ]);

  useIntervalRun(fetchPerfProfiles);

  function fetchPerfProfiles(page?: number) {
    dispatch({
      type: 'misc/fetchPerfProfiles',
      payload: {
        page,
      },
    });
  }

  function deletePerfProfile(record: IPerfProfile) {
    Modal.confirm({
      title: '删除报告？',
      content: '你确定要删除这个报告吗？删除后不可恢复',
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
    setUploadRemoteModalVisible(true);
    setUploadRemoteUrl(`/perfprofiles/${record.uuid}`);
  }

  function handleAddPerfProfile(instanceId: string, node: string): Promise<any> {
    return new Promise((resolve, reject) => {
      dispatch({
        type: 'misc/addPerfProfile',
        payload: { instanceId, node },
      }).then((val: any) => resolve());
    });
  }

  function handleTableChange(curPagination: PaginationConfig) {
    fetchPerfProfiles(curPagination.current as number);
  }

  function handleLocalFileUploaded(res: IPerfProfile) {
    dispatch({
      type: 'misc/savePerfProfile',
      payload: res,
    });
  }

  function showAddReportModal() {
    setAddReportModalVisible(true);
    if (instances.length === 0) {
      dispatch({
        type: 'inspection/fetchInstances',
      });
    }
  }

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h2>火焰图 &amp; Profile 报告列表</h2>
        {curUser.role === 'admin' && (
          <Button type="primary" onClick={showAddReportModal}>
            + 获取
          </Button>
        )}
        {curUser.role === 'dba' && (
          <Button type="primary" onClick={() => setUploadLocalModalVisible(true)}>
            + 导入本地报告
          </Button>
        )}
      </div>
      <Table
        loading={loading}
        dataSource={perfprofile.list}
        columns={columns}
        onChange={handleTableChange}
        pagination={pagination}
      />
      <AddMiscReportModal
        instances={instances}
        visible={addReportModalVisble}
        onClose={() => setAddReportModalVisible(false)}
        onData={handleAddPerfProfile}
      />
      <UploadRemoteReportModal
        visible={uploadRemoteModalVisible}
        onClose={() => setUploadRemoteModalVisible(false)}
        uploadUrl={uploadRemoteUrl}
      />
      <UploadLocalReportModal
        visible={uploadLocalModalVisible}
        onClose={() => setUploadLocalModalVisible(false)}
        actionUrl="/api/v1/perfprofiles"
        onData={handleLocalFileUploaded}
      />
    </div>
  );
}

export default connect(({ user, misc, inspection, loading }: ConnectState) => ({
  curUser: user.currentUser,
  perfprofile: misc.perfprofile,
  instances: inspection.instances,
  loading: loading.effects['misc/fetchPerfProfiles'],
}))(PerfProfileList);
