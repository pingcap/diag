import React, { useEffect, useMemo, useState } from 'react';
import { Table, Button, Tooltip, Icon, Divider, Modal } from 'antd';
import { connect } from 'dva';
import { Link } from 'umi';
import { ConnectState, ConnectProps, InspectionModelState, Dispatch } from '@/models/connect';
import { IFormatInstance, IInstance } from '@/models/inspection';
import AddInstanceModal from '@/components/AddInstanceModal';

const styles = require('./InstanceList.less');

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
    render: (text: any, record: IFormatInstance) => (
      <Link to={`/inspection/instances/${record.uuid}/reports`}>{text}</Link>
    ),
  },
  {
    title: 'PD 址址:端口',
    dataIndex: 'pd',
    key: 'pd',
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
      if (record.message) {
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
        <Link to={`/inspection/instances/${record.uuid}/reports`}>查看</Link>
        <Divider type="vertical" />
        <a href="#">设置</a>
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
}

function InstanceList({ inspection, dispatch }: InstanceListProps) {
  const [modalVisible, setModalVisible] = useState(false);
  useEffect(() => {
    dispatch({ type: 'inspection/fetchInstances' });
  }, []);

  const columns = useMemo(() => tableColumns(onDelete), []);

  function onDelete(record: IFormatInstance) {
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
    setModalVisible(true);
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
      <Table dataSource={inspection.instances} columns={columns} pagination={false} />
      <AddInstanceModal
        visible={modalVisible}
        onClose={() => setModalVisible(false)}
        onData={addInstance}
      />
    </div>
  );
}

export default connect(({ inspection }: ConnectState) => ({ inspection }))(InstanceList);
