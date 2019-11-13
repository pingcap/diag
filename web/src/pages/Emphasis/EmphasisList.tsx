import React, { useMemo, useState, useEffect } from 'react';
import { Table, Button, Divider, Modal, Tooltip, Icon, Select, DatePicker } from 'antd';
import { connect } from 'dva';
import { Link } from 'umi';
import { PaginationConfig } from 'antd/lib/table';
import { RangePickerValue } from 'antd/lib/date-picker/interface';
import moment from 'moment';
import UploadRemoteReportModal from '@/components/UploadRemoteReportModal';
import UploadLocalReportModal from '@/components/UploadLocalReportModal';
import { ConnectState, ConnectProps, Dispatch, EmphasisModelState } from '@/models/connect';
import { CurrentUser } from '@/models/user';
import { IFormatInstance } from '@/models/inspection';
import { IEmphasis } from '@/models/emphasis';
import { formatDatetime } from '@/utils/datetime-util';
import { useIntervalRun } from '@/custom-hooks/use-interval-run';

const { Option } = Select;

const styles = require('../style.less');

function getReportDetailLink(instanceId: string | undefined, reportId: string) {
  return instanceId === undefined
    ? `/inspection/emphasis/${reportId}`
    : `/inspection/instances/${instanceId}/emphasis/${reportId}`;
}

const tableColumns = (
  curUser: CurrentUser,
  onDelete: any,
  onUpload: any,
  instanceId: string | undefined,
) => {
  const columns = [
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
      title: '排查时间',
      key: 'emphasis_time',
      render: (text: any, record: IEmphasis) => (
        <span>
          {formatDatetime(record.investgating_start)} ~ {formatDatetime(record.investgating_end)}
        </span>
      ),
    },
    {
      title: '创建时间',
      key: 'create_time',
      render: (text: any, record: IEmphasis) => {
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
        return <span>{formatDatetime(record.created_time)}</span>;
      },
    },
    {
      title: '操作',
      key: 'action',
      render: (text: any, record: IEmphasis) => (
        <span>
          {record.status === 'success' ? (
            <Link to={getReportDetailLink(instanceId, record.uuid)}>详情</Link>
          ) : (
            <span>详情</span>
          )}
          {curUser.role === 'admin' && (
            <React.Fragment>
              <Divider type="vertical" />
              {record.status === 'success' ? (
                <a download href={`/api/v1/emphasis/zip/${record.uuid}.tar.gz`}>
                  下载
                </a>
              ) : (
                <span>下载</span>
              )}
              {curUser.ka && (
                <React.Fragment>
                  <Divider type="vertical" />
                  {record.status === 'success' ? (
                    <a onClick={() => onUpload(record)}>上传</a>
                  ) : (
                    <span>上传</span>
                  )}
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
  return columns;
};

// TODO: extract
const endOfToday = moment().endOf('day');
const EMPHASIS_PROBLEMS = [
  {
    value: 'db',
    text: '数据库瓶颈分析',
  },
];
// ['数据库瓶颈分析', 'QPS 抖动问题', '.99 延迟高问题'];

interface ReportListProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
  instances: IFormatInstance[];
  emphasis: EmphasisModelState;

  loading: boolean;
  loadingInstances: boolean;
}

function ReportList({
  dispatch,
  curUser,
  instances,
  emphasis,
  match,
  loading,
  loadingInstances,
}: ReportListProps) {
  const initInstanceId: string | undefined = match && match.params && (match.params as any).id;

  // TODO: try to use useReducer to replace so many useState
  const [selectedInstance, setSelectedInstance] = useState(initInstanceId);

  const [uploadRemoteModalVisible, setUploadRemoteModalVisible] = useState(false);
  const [remoteUploadUrl, setRemoteUploadUrl] = useState('');

  const [uploadLocalModalVisible, setUploadLocalModalVisible] = useState(false);

  const [issue, setIssue] = useState('');

  const defTimeRange: [moment.Moment, moment.Moment] = [moment().subtract(1, 'hours'), moment()];
  const [timeRange, setTimeRange] = useState<[string, string]>([
    defTimeRange[0].format(),
    defTimeRange[1].format(),
  ]);

  function disableDate(current: moment.Moment | undefined) {
    // Can not select days before today
    return (current && current > endOfToday) || false;
  }

  const pagination: PaginationConfig = useMemo(
    () => ({
      total: emphasis.emphasis.total,
      current: emphasis.emphasis.cur_page,
    }),
    [emphasis.emphasis.cur_page, emphasis.emphasis.total],
  );

  useIntervalRun(fetchEmphasisList, 10 * 1000, [selectedInstance]);

  useEffect(() => {
    fetchInstances();
  }, []);

  function fetchInstances() {
    dispatch({
      type: 'inspection/fetchInstances',
    });
  }

  function fetchEmphasisList(page?: number) {
    return dispatch({
      type: 'emphasis/fetchEmphasisList',
      payload: {
        page,
        instanceId: selectedInstance,
      },
    });
  }

  const columns = useMemo(
    () => tableColumns(curUser, deleteEmphasis, uploadEmphasis, selectedInstance),
    [curUser, selectedInstance],
  );

  function inspectEmphasis() {
    dispatch({
      type: 'emphasis/addEmphasis',
      payload: {
        instanceId: selectedInstance,
        investgating_start: timeRange[0],
        investgating_end: timeRange[1],
        investgating_problem: issue,
      },
    });
  }

  function deleteEmphasis(record: IEmphasis) {
    Modal.confirm({
      title: '删除报告？',
      content: '你确定要删除这个报告吗？删除后不可恢复',
      okText: '删除',
      okButtonProps: { type: 'danger' },
      onOk() {
        dispatch({
          type: 'emphasis/deleteEmphasis',
          payload: record.uuid,
        });
      },
    });
  }

  function uploadEmphasis(record: IEmphasis) {
    setUploadRemoteModalVisible(true);
    setRemoteUploadUrl(`/emphasis/${record.uuid}`);
  }

  function handleTableChange(curPagination: PaginationConfig) {
    fetchEmphasisList(curPagination.current as number);
  }

  function handleLocalFileUploaded(res: IEmphasis) {
    dispatch({
      type: 'emphasis/saveEmphasis',
      payload: res,
    });
  }

  function handleRangeChange(dates: RangePickerValue, dateStrings: [string, string]) {
    // 如果用户进行了 clear，dates 为 [], dateStrings 为 ["", ""]
    if (dates[0] && dates[1]) {
      setTimeRange([dates[0].format(), dates[1].format()]);
    } else {
      setTimeRange(['', '']);
    }
  }

  function handleIssueChange(value: string | undefined) {
    setIssue(value || '');
  }

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h2>报告列表</h2>
        {curUser.role === 'admin' && (
          <Select
            value={selectedInstance}
            loading={loadingInstances}
            allowClear
            placeholder="选择集群实例"
            style={{ width: 200, marginLeft: 12 }}
            onChange={(val: any) => setSelectedInstance(val)}
          >
            {instances.map(item => (
              <Option value={item.uuid} key={item.uuid}>
                {item.name}
              </Option>
            ))}
          </Select>
        )}
        {curUser.role === 'admin' && (
          <Select
            allowClear
            placeholder="请选择重点问题"
            style={{ width: 140, marginRight: 12, marginLeft: 12 }}
            onChange={handleIssueChange}
          >
            {EMPHASIS_PROBLEMS.map(item => (
              <Option value={item.value} key={item.value}>
                {item.text}
              </Option>
            ))}
          </Select>
        )}
        {curUser.role === 'admin' && selectedInstance !== undefined && (
          <React.Fragment>
            <DatePicker.RangePicker
              style={{ marginRight: 12 }}
              showTime={{ format: 'HH:mm' }}
              format="YYYY-MM-DD HH:mm"
              placeholder={['诊断起始时间', '结束时间']}
              onChange={handleRangeChange}
              defaultValue={defTimeRange}
              disabledDate={disableDate}
            />
            <Button
              type="primary"
              disabled={timeRange[0] === '' || issue === ''}
              onClick={inspectEmphasis}
            >
              开始诊断
            </Button>
          </React.Fragment>
        )}
        <div className={styles.space} />
        {curUser.role === 'dba' && (
          <Button type="primary" onClick={() => setUploadLocalModalVisible(true)}>
            + 导入本地报告
          </Button>
        )}
      </div>
      <Table
        loading={loading}
        dataSource={emphasis.emphasis.list}
        columns={columns}
        onChange={handleTableChange}
        pagination={pagination}
      />
      <UploadRemoteReportModal
        visible={uploadRemoteModalVisible}
        onClose={() => setUploadRemoteModalVisible(false)}
        uploadUrl={remoteUploadUrl}
      />
      <UploadLocalReportModal
        visible={uploadLocalModalVisible}
        onClose={() => setUploadLocalModalVisible(false)}
        actionUrl="/api/v1/emphasis"
        onData={handleLocalFileUploaded}
      />
    </div>
  );
}

export default connect(({ user, inspection, emphasis, loading }: ConnectState) => ({
  curUser: user.currentUser,
  instances: inspection.instances,
  emphasis,
  loading: loading.effects['emphasis/fetchEmphasisList'],
  loadingInstances: loading.effects['inspection/fetchInstances'],
}))(ReportList);
