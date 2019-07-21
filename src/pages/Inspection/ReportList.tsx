import React, { useEffect, useMemo, useState } from 'react';
import { Table, Button, Divider, Modal } from 'antd';
import { connect } from 'dva';
import { Link } from 'umi';
import { PaginationConfig } from 'antd/lib/table';
import { ConnectState, ConnectProps, InspectionModelState, Dispatch } from '@/models/connect';
import { IInspection, IFormatInspection } from '@/models/inspection';

const styles = require('./style.less');

const tableColumns = (onDelete: any, onCopy: any) => [
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
        <Link to={`/inspection/reports/${record.uuid}`}>查看</Link>
        <Divider type="vertical" />
        <a onClick={() => onCopy(record)}>拷贝</a>
        <Divider type="vertical" />
        <a style={{ color: 'red' }} onClick={() => onDelete(record)}>
          删除
        </a>
      </span>
    ),
  },
];

interface ReportListProps extends ConnectProps {
  inspection: InspectionModelState;
  dispatch: Dispatch;
  loading: boolean;
}

function ReportList({ inspection, dispatch, match, loading }: ReportListProps) {
  const [curInspection, setCurInspection] = useState<IInspection | null>(null);

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

  const columns = useMemo(() => tableColumns(deleteInspection, copyInsepction), []);

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
      onCancel() {},
    });
  }

  function copyInsepction(record: IFormatInspection) {
    setCurInspection(record);
  }

  function manuallyInspect() {
    // TODO:
    // post inspections
  }

  function handleTableChange(curPagination: PaginationConfig) {
    fetchInspections(curPagination.current as number);
  }

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h2>诊断报告列表</h2>
        <Button type="primary" onClick={manuallyInspect}>
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
    </div>
  );
}

export default connect(({ inspection, loading }: ConnectState) => ({
  inspection,
  loading: loading.effects['inspection/fetchInspections'],
}))(ReportList);
