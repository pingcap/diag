import React, { useMemo, useState } from 'react';
import { Table, Button, Tooltip, Icon, Divider, Modal, Menu, Dropdown } from 'antd';
import { connect } from 'dva';
import { Link } from 'umi';
import { ConnectState, ConnectProps, InspectionModelState, Dispatch } from '@/models/connect';
import { IFormatInstance, IInstance } from '@/models/inspection';
import AddInstanceModal from '@/components/AddInstanceModal';
import { useIntervalRun } from '@/custom-hooks/use-interval-run';

const styles = require('../style.less');

const tableColumns = (onDelete: any) => [
  {
    title: '用户名',
    dataIndex: 'user',
    key: 'user',
  },
  {
    title: '实例名',
    dataIndex: 'name',
    key: 'name',
    render: (text: any, record: IFormatInstance) => {
      if (text === '') {
        return <span>获取中...</span>;
      }
      return <Link to={`/inspection/instances/${record.uuid}/reports`}>{text}</Link>;
    },
  },
  {
    title: 'PD 地址:端口',
    dataIndex: 'pd',
    key: 'pd',
    render: (text: any, record: IFormatInstance) => {
      if (text === '' && record.status === 'pending') {
        return <span>获取中...</span>;
      }
      return <span>{text || 'unknown'}</span>;
    },
  },
  {
    title: '创建时间',
    dataIndex: 'format_create_time',
    key: 'format_create_time',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    render: (text: any, record: IFormatInstance) => {
      if (record.status === 'exception') {
        return (
          <div className={styles.instance_status}>
            <span style={{ color: 'red' }}>{text}</span>
            <Tooltip title={record.message}>
              <Icon type="question-circle" />
            </Tooltip>
          </div>
        );
      }
      return <span>{text}</span>;
    },
  },
  {
    title: '操作',
    key: 'action',
    render: (text: any, record: IFormatInstance) => (
      <span>
        {record.status === 'success' ? (
          <Dropdown
            trigger={['click']}
            overlay={
              <Menu>
                <Menu.Item>
                  <Link to={`/inspection/instances/${record.uuid}/reports`}>诊断报告</Link>
                </Menu.Item>
                <Menu.Item>
                  <Link to={`/inspection/instances/${record.uuid}/emphasis`}>重点问题</Link>
                </Menu.Item>
                <Menu.Item>
                  <Link to="/inspection/perfprofiles">火焰图 & profile</Link>
                </Menu.Item>
              </Menu>
            }
          >
            <a href="#">诊断管理</a>
          </Dropdown>
        ) : (
          <span>诊断管理</span>
        )}
        <Divider type="vertical" />
        <a style={{ color: 'red' }} onClick={() => onDelete(record)}>
          删除
        </a>
      </span>
    ),
  },
];

interface InstanceListProps extends ConnectProps {
  inspection: InspectionModelState;
  dispatch: Dispatch;
  loading: boolean;
}

function InstanceList({ inspection, dispatch, loading }: InstanceListProps) {
  const [addModalVisible, setAddModalVisible] = useState(false);

  useIntervalRun(() => dispatch({ type: 'inspection/fetchInstances' }));

  const columns = useMemo(() => tableColumns(deleteInstance), []);

  function deleteInstance(record: IFormatInstance) {
    Modal.confirm({
      title: '删除实例？',
      content: '你确定要删除这个实例吗？删除后不可恢复',
      okText: '删除',
      okButtonProps: { type: 'danger' },
      onOk() {
        dispatch({
          type: 'inspection/deleteInstance',
          payload: record.uuid,
        });
      },
      onCancel() {},
    });
  }

  function onAdd() {
    setAddModalVisible(true);
  }

  function addInstance(instance: IInstance) {
    // sync action
    dispatch({
      type: 'inspection/saveInstance',
      payload: instance,
    });
  }

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h2>集群实例列表</h2>
        <Button type="primary" onClick={onAdd}>
          +添加实例
        </Button>
      </div>
      <Table
        dataSource={inspection.instances}
        columns={columns}
        pagination={false}
        loading={loading}
      />
      <AddInstanceModal
        visible={addModalVisible}
        onClose={() => setAddModalVisible(false)}
        onData={addInstance}
      />
    </div>
  );
}

export default connect(({ inspection, loading }: ConnectState) => ({
  inspection,
  loading: loading.effects['inspection/fetchInstances'],
}))(InstanceList);
