import React, { useEffect, useMemo, useState } from 'react';
import { Table, Button, Tooltip, Icon, Divider, Modal } from 'antd';
import { connect } from 'dva';
import { Link } from 'umi';
import { ConnectState, ConnectProps, InspectionModelState, Dispatch } from '@/models/connect';
import { IFormatInstance, IInstance } from '@/models/inspection';
import AddInstanceModal from '@/components/AddInstanceModal';
import ConfigInstanceModal from '@/components/ConfigInstanceModal';

const styles = require('../style.less');

const tableColumns = (onDelete: any, onConfig: any) => [
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
    title: 'PD 址址:端口',
    dataIndex: 'pd',
    key: 'pd',
    render: (text: any, record: IFormatInstance) => <span>{text || '获取中...'}</span>,
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
          <Link to={`/inspection/instances/${record.uuid}/reports`}>查看</Link>
        ) : (
          <span>查看</span>
        )}
        <Divider type="vertical" />
        {record.status === 'success' ? (
          <a onClick={() => onConfig(record)}>设置</a>
        ) : (
          <span>设置</span>
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
  const [configModalVisible, setConfigModalVisible] = useState(false);
  const [curInstance, setCurInstance] = useState<IInstance | null>(null);

  useEffect(() => {
    dispatch({ type: 'inspection/fetchInstances' });
  }, []);

  const columns = useMemo(() => tableColumns(deleteInstance, configInstance), []);

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

  function configInstance(record: IFormatInstance) {
    setConfigModalVisible(true);
    setCurInstance(record);
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

  function closeConfigModal() {
    setConfigModalVisible(false);
    setCurInstance(null);
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
      <ConfigInstanceModal
        visible={configModalVisible}
        onClose={closeConfigModal}
        manual={false}
        instanceId={curInstance ? curInstance.uuid : ''}
      />
    </div>
  );
}

export default connect(({ inspection, loading }: ConnectState) => ({
  inspection,
  loading: loading.effects['inspection/fetchInstances'],
}))(InstanceList);
