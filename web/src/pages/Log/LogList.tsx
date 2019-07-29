import React, { useEffect, useState } from 'react';
import { Table, Button, DatePicker, Input, Select, Modal } from 'antd';
import { connect } from 'dva';
import { RangePickerValue } from 'antd/lib/date-picker/interface';
import { ConnectState, ConnectProps, Dispatch } from '@/models/connect';
import { LogModelState } from '@/models/log';
import UploadRemoteReportModal from '@/components/UploadRemoteReportModal';
import { CurrentUser } from '@/models/user';
import UploadLocalReportModal from '@/components/UploadLocalReportModal';

const { Option } = Select;

const styles = require('../style.less');

const tableColumns = [
  {
    title: '时间',
    dataIndex: 'time',
    key: 'time',
  },
  {
    title: '实例名称',
    dataIndex: 'instance_name',
    key: 'instance_name',
  },
  {
    title: '日志级别',
    dataIndex: 'level',
    key: 'level',
  },
  {
    title: '内容',
    dataIndex: 'content',
    key: 'content',
  },
];

const logLevels = ['ALL', 'DEBUG', 'INFO', 'WARNING', 'ERROR', 'OTHERS'];

interface ReportListProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
  log: LogModelState;
  searchingLogs: boolean;
  loadingMoreLogs: boolean;
  loadingLogInstances: boolean;
}

function ReportList({
  dispatch,
  curUser,
  log,
  searchingLogs,
  loadingMoreLogs,
  loadingLogInstances,
}: ReportListProps) {
  const [logLevel, setLogLevel] = useState('');
  const [selectedInstanceId, setSelectedInstanceId] = useState('');
  const [timeRange, setTimeRange] = useState<[string, string]>(['', '']);

  const [uploadRemoteModalVisible, setUploadRemoteModalVisible] = useState(false);
  const [uploadLocalModalVisible, setUploadLocalModalVisible] = useState(false);

  // for dba user
  const [logId, setLogId] = useState('');

  useEffect(() => {
    dispatch({ type: 'log/fetchLogInstances' });
    dispatch({ type: 'log/resetLogs' });
  }, []);

  function handleLogLevelChange(value: string | undefined) {
    setLogLevel(value || '');
  }

  function handleInstanceChange(value: string | undefined) {
    // 如果用户进行了 clear，value 为 undefined
    setSelectedInstanceId(value || '');
  }

  function handleRangeChange(dates: RangePickerValue, dateStrings: [string, string]) {
    // 如果用户进行了 clear，dates 为 [], dateStrings 为 ["", ""]
    if (dates[0] && dates[1]) {
      // console.log(dates[0].format());
      setTimeRange([dates[0].format(), dates[1].format()]);
    } else {
      setTimeRange(['', '']);
    }
  }

  function handleSearch(value: string) {
    dispatch({
      type: 'log/searchLogs',
      payload: {
        logInstanceId: selectedInstanceId,
        logId,
        startTime: timeRange[0],
        endTime: timeRange[1],
        search: value,
        logLevel,
      },
    });
  }

  function handleDownlaod() {
    Modal.confirm({ title: '下载搜索结果', content: 'TODO' });
  }

  function handleLoadMore() {
    dispatch({
      type: 'log/loadMoreLogs',
      payload: {
        logInstanceId: selectedInstanceId,
        logId,
      },
    });
  }

  function handleLocalFileUploaded(res: any) {
    // TODO
    console.log(res.logId);
    setLogId(res.logId);
  }

  function disableSearch(): boolean {
    if (curUser.role === 'admin') {
      return logLevel === '' || selectedInstanceId === '' || timeRange[0] === '';
    }
    if (curUser.role === 'dba') {
      return logLevel === '' || logId === '';
    }
    return false;
  }

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h2>日志搜索</h2>
        <div className={styles.space} />
        {curUser.role === 'admin' && curUser.ka && (
          <Button
            type="primary"
            style={{ marginRight: 20 }}
            onClick={() => setUploadRemoteModalVisible(true)}
            disabled={log.logs.length === 0}
          >
            上传
          </Button>
        )}
        {curUser.role === 'admin' && (
          <Button type="primary" onClick={handleDownlaod} disabled={log.logs.length === 0}>
            下载
          </Button>
        )}
        {curUser.role === 'dba' && (
          <Button type="primary" onClick={() => setUploadLocalModalVisible(true)}>
            + 上传本地报告
          </Button>
        )}
      </div>
      <div className={styles.list_header}>
        {curUser.role === 'dba' && logId === '' && <span>请先上传本地日志文件</span>}
        {(curUser.role === 'admin' || logId !== '') && (
          <Select
            allowClear
            placeholder="选择日志级别"
            style={{ width: 140, marginRight: 20 }}
            onChange={handleLogLevelChange}
          >
            {logLevels.map(item => (
              <Option value={item} key={item}>
                {item}
              </Option>
            ))}
          </Select>
        )}
        {curUser.role === 'admin' && (
          <React.Fragment>
            <Select
              loading={loadingLogInstances}
              allowClear
              placeholder="选择集群实例"
              style={{ width: 140 }}
              onChange={handleInstanceChange}
            >
              {log.logInstances.map(item => (
                <Option value={item.uuid} key={item.uuid}>
                  {item.name}
                </Option>
              ))}
            </Select>
            <DatePicker.RangePicker
              style={{ marginLeft: 12, marginRight: 12 }}
              showTime={{ format: 'HH:mm' }}
              format="YYYY-MM-DD HH:mm"
              placeholder={['起始时间', '结束时间']}
              onChange={handleRangeChange}
            />
          </React.Fragment>
        )}
        {(curUser.role === 'admin' || logId !== '') && (
          <Input.Search
            allowClear
            disabled={disableSearch()}
            placeholder="search"
            onSearch={handleSearch}
            style={{ width: 200, height: 32 }}
            size="small"
          />
        )}
        <div className={styles.space} />
      </div>
      <br />
      <Table
        loading={searchingLogs}
        dataSource={log.logs}
        columns={tableColumns}
        pagination={false}
      />
      {log.token && (
        <Button onClick={handleLoadMore} loading={loadingMoreLogs} className={styles.load_more_btn}>
          加载更多
        </Button>
      )}
      <UploadRemoteReportModal
        content="确定要将此次搜索结果上传吗？"
        visible={uploadRemoteModalVisible}
        onClose={() => setUploadRemoteModalVisible(false)}
        uploadUrl={`/loginstances/${selectedInstanceId}/logs`}
      />
      <UploadLocalReportModal
        title="上传本地日志"
        visible={uploadLocalModalVisible}
        onClose={() => setUploadLocalModalVisible(false)}
        actionUrl="/api/v1/logs"
        onData={handleLocalFileUploaded}
      />
    </div>
  );
}

export default connect(({ user, log, loading }: ConnectState) => ({
  curUser: user.currentUser,
  log,
  searchingLogs: loading.effects['log/searchLogs'],
  loadingMoreLogs: loading.effects['log/loadMoreLogs'],
  loadingLogInstances: loading.effects['log/fetchLogInstances'],
}))(ReportList);
