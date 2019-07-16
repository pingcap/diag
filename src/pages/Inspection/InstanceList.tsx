import React, { useEffect, useMemo } from 'react';
import { Table, Button, Tooltip, Icon, Divider } from 'antd';
import { connect } from 'dva';
import moment from 'moment';
import { Link } from 'umi';
import { ConnectState, ConnectProps, InspectionModelState, Dispatch } from '@/models/connect';
import { IInstance } from '@/models/inspection';

const styles = require('./InstanceList.less');

const columns = [
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
        <a href="#" style={{ color: 'red' }}>
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

interface IFormatInstance extends IInstance {
  user: string;
  key: string;
  format_create_time: string;
}

function convertInstances(instances: IInstance[]): IFormatInstance[] {
  return instances.map(item => ({
    ...item,
    user: 'default',
    key: item.uuid,
    format_create_time: moment(item.create_time).format('YYYY-MM-DD hh:mm'),
  }));
}

function InstanceList({ inspection, dispatch }: InstanceListProps) {
  useEffect(() => {
    dispatch({ type: 'inspection/fetchInstances' });
  }, []);

  const dataSource = useMemo(() => convertInstances(inspection.instances), [inspection.instances]);

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h2>集群实例列表</h2>
        <Button type="primary">+添加实例</Button>
      </div>
      <Table dataSource={dataSource} columns={columns} pagination={false} />
    </div>
  );
}

export default connect(({ inspection }: ConnectState) => ({ inspection }))(InstanceList);
