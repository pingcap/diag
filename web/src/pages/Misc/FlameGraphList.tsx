import React, { useEffect, useMemo } from 'react';
import { Table, Button, Divider, Modal } from 'antd';
import { connect } from 'dva';
import { Link } from 'umi';
import { PaginationConfig } from 'antd/lib/table';
import { ConnectState, ConnectProps, Dispatch } from '@/models/connect';
import { IFlameGraphInfo, IFlameGraph } from '@/models/misc';

const styles = require('../style.less');

const tableColumns = (onDelete: any) => [
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
    title: '报告保存地址',
    dataIndex: 'report_path',
    key: 'report_path',
  },
  {
    title: '操作',
    key: 'action',
    render: (text: any, record: IFlameGraph) => (
      <span>
        <Link to={`/misc/flamegraphs/${record.uuid}`}>查看</Link>
        <Divider type="vertical" />
        <a download href={`/api/v1/flamegraphs/${record.uuid}.tar.gz`}>
          拷贝
        </a>
        <Divider type="vertical" />
        <a style={{ color: 'red' }} onClick={() => onDelete(record)}>
          删除
        </a>
      </span>
    ),
  },
];

interface FlameGraphListProps extends ConnectProps {
  flamegraph: IFlameGraphInfo;
  dispatch: Dispatch;
  loading: boolean;
  collecting: boolean;
}

function FlameGraphList({ flamegraph, dispatch, loading, collecting }: FlameGraphListProps) {
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

  const columns = useMemo(() => tableColumns(deleteFlamegraph), []);

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

  function generateFlamegraph() {
    Modal.confirm({
      title: '收集火焰图？',
      content: '你确定要收集火焰图吗？',
      okText: '收集',
      onOk() {
        dispatch({
          type: 'misc/addFlamegraph',
        });
      },
    });
  }

  function handleTableChange(curPagination: PaginationConfig) {
    fetchFlamegraphs(curPagination.current as number);
  }

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h2>火焰图报告列表</h2>
        <Button type="primary" onClick={generateFlamegraph} loading={collecting}>
          + 获取
        </Button>
      </div>
      <Table
        loading={loading}
        dataSource={flamegraph.list}
        columns={columns}
        onChange={handleTableChange}
        pagination={pagination}
      />
    </div>
  );
}

export default connect(({ misc, loading }: ConnectState) => ({
  flamegraph: misc.flamegraph,
  loading: loading.effects['misc/fetchFlamegraphs'],
  collecting: loading.effects['misc/addFlamegraph'],
}))(FlameGraphList);
