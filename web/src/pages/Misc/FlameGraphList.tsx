import React, { useEffect, useMemo, useState } from 'react';
import { Table, Button, Divider, Modal } from 'antd';
import { connect } from 'dva';
import { Link } from 'umi';
import { PaginationConfig } from 'antd/lib/table';
import { ConnectState, ConnectProps, Dispatch } from '@/models/connect';
import { IFlameGraphInfo, IFlameGraph } from '@/models/misc';
import AddMiscReportModal from '@/components/AddMiscReportModal';
import UploadRemoteReportModal from '@/components/UploadRemoteReportModal';
import { CurrentUser } from '@/models/user';
import UploadLocalReportModal from '@/components/UploadLocalReportModal';
import { IFormatInstance } from '@/models/inspection';

const styles = require('../style.less');

const tableColumns = (curUser: CurrentUser, onDelete: any, onUpload: any) => [
  {
    title: '火焰图报告 ID',
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
    render: (text: any, record: IFlameGraph) => {
      if (record.status === 'running') {
        return <span>running...</span>;
      }
      return <span>{text}</span>;
    },
  },
  {
    title: '操作',
    key: 'action',
    render: (text: any, record: IFlameGraph) => (
      <span>
        {record.status === 'running' ? (
          <span>详情</span>
        ) : (
          <Link to={`/misc/flamegraphs/${record.uuid}`}>详情</Link>
        )}
        {curUser.role === 'admin' && (
          <React.Fragment>
            <Divider type="vertical" />
            {record.status === 'running' ? (
              <span>下载</span>
            ) : (
              <a download href={`/api/v1/flamegraphs/${record.uuid}.tar.gz`}>
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

interface FlameGraphListProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
  flamegraph: IFlameGraphInfo;
  instances: IFormatInstance[];
  loading: boolean;
}

function FlameGraphList({
  dispatch,
  curUser,
  flamegraph,
  instances,
  loading,
}: FlameGraphListProps) {
  const [addReportModalVisible, setAddReportModalVisible] = useState(false);

  const [uploadRemoteModalVisible, setUploadRemoteModalVisible] = useState(false);
  const [uploadRemoteUrl, setUploadRemoteUrl] = useState('');

  const [uploadLocalModalVisible, setUploadLocalModalVisible] = useState(false);

  const pagination: PaginationConfig = useMemo(
    () => ({
      total: flamegraph.total,
      current: flamegraph.cur_page,
    }),
    [flamegraph.total, flamegraph.cur_page],
  );

  useEffect(() => {
    fetchFlamegraphs(flamegraph.cur_page);
  }, []);

  function fetchFlamegraphs(page: number) {
    dispatch({
      type: 'misc/fetchFlamegraphs',
      payload: {
        page,
      },
    });
  }

  const columns = useMemo(() => tableColumns(curUser, deleteFlamegraph, uploadFlamegraph), [
    curUser,
  ]);

  function deleteFlamegraph(record: IFlameGraph) {
    Modal.confirm({
      title: '删除火焰图报告？',
      content: '你确定要删除这个火焰图报告吗？删除后不可恢复',
      okText: '删除',
      okButtonProps: { type: 'danger' },
      onOk() {
        dispatch({
          type: 'misc/deleteFlamegraph',
          payload: record.uuid,
        });
      },
    });
  }

  function uploadFlamegraph(record: IFlameGraph) {
    setUploadRemoteModalVisible(true);
    setUploadRemoteUrl(`/api/v1/flamegraphs/${record.uuid}`);
  }

  function handleAddFlamegraph(instanceId: string): Promise<any> {
    return new Promise((resolve, reject) => {
      dispatch({
        type: 'misc/addFlamegraph',
        payload: instanceId,
      }).then((val: any) => resolve());
    });
  }

  function handleTableChange(curPagination: PaginationConfig) {
    fetchFlamegraphs(curPagination.current as number);
  }

  function handleLocalFileUploaded(res: IFlameGraph) {
    dispatch({
      type: 'misc/saveFlamegraph',
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
        <h2>火焰图报告列表</h2>
        {curUser.role === 'admin' && (
          <Button type="primary" onClick={showAddReportModal}>
            + 获取
          </Button>
        )}
        {curUser.role === 'dba' && (
          <Button type="primary" onClick={() => setUploadLocalModalVisible(true)}>
            + 上传本地报告
          </Button>
        )}
      </div>
      <Table
        loading={loading}
        dataSource={flamegraph.list}
        columns={columns}
        onChange={handleTableChange}
        pagination={pagination}
      />
      <AddMiscReportModal
        instances={instances}
        visible={addReportModalVisible}
        onClose={() => setAddReportModalVisible(false)}
        onData={handleAddFlamegraph}
      />
      <UploadRemoteReportModal
        visible={uploadRemoteModalVisible}
        onClose={() => setUploadRemoteModalVisible(false)}
        uploadUrl={uploadRemoteUrl}
      />
      <UploadLocalReportModal
        visible={uploadLocalModalVisible}
        onClose={() => setUploadLocalModalVisible(false)}
        actionUrl="/api/v1/flamegraphs"
        onData={handleLocalFileUploaded}
      />
    </div>
  );
}

export default connect(({ user, misc, inspection, loading }: ConnectState) => ({
  curUser: user.currentUser,
  flamegraph: misc.flamegraph,
  instances: inspection.instances,
  loading: loading.effects['misc/fetchFlamegraphs'],
}))(FlameGraphList);
