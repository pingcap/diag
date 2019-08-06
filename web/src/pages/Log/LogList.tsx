import React, { useEffect, useState } from 'react';
import { Table, Button, DatePicker, Input, Select } from 'antd';
import { connect } from 'dva';
import { RangePickerValue } from 'antd/lib/date-picker/interface';
import { ConnectState, ConnectProps, Dispatch } from '@/models/connect';
import { LogModelState, ILogFile } from '@/models/log';
import UploadRemoteReportModal from '@/components/UploadRemoteReportModal';
import { CurrentUser } from '@/models/user';
import UploadLocalReportModal from '@/components/UploadLocalReportModal';

const { Option } = Select;

const styles = require('../style.less');

const tableColumns = [
  {
    title: '时间',
    dataIndex: 'format_time',
    key: 'format_time',
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

const logLevels = ['DEBUG', 'INFO', 'WARNING', 'ERROR', 'OTHERS'];

interface ReportListProps extends ConnectProps {
  dispatch: Dispatch;

  curUser: CurrentUser;
  log: LogModelState;
  searchingLogs: boolean;
  loadingMoreLogs: boolean;
  loadingLogInstances: boolean;
  loadingLogFiles: boolean;
}

function ReportList({
  dispatch,
  curUser,
  log,
  searchingLogs,
  loadingMoreLogs,
  loadingLogInstances,
  loadingLogFiles,
}: ReportListProps) {
  // for admin
  const [logInstanceId, setLogInstanceId] = useState<string | undefined>(undefined);
  // for dba user
  const [logFileId, setLogFileId] = useState<string | undefined>(undefined);

  const [timeRange, setTimeRange] = useState<[string, string]>(['', '']);
  const [searchStr, setSearchStr] = useState('');
  const [logLevel, setLogLevel] = useState('');

  const [uploadRemoteModalVisible, setUploadRemoteModalVisible] = useState(false);
  const [uploadLocalModalVisible, setUploadLocalModalVisible] = useState(false);

  useEffect(() => {
    if (curUser.role === 'admin') {
      dispatch({ type: 'log/fetchLogInstances' });
    }
    if (curUser.role === 'dba') {
      dispatch({ type: 'log/fetchLogFiles' });
    }
    dispatch({ type: 'log/resetLogs' });
  }, [curUser]);

  function handleLogLevelChange(value: string | undefined) {
    setLogLevel(value || '');
  }

  function handleRangeChange(dates: RangePickerValue, dateStrings: [string, string]) {
    // 如果用户进行了 clear，dates 为 [], dateStrings 为 ["", ""]
    if (dates[0] && dates[1]) {
      setTimeRange([dates[0].format(), dates[1].format()]);
    } else {
      setTimeRange(['', '']);
    }
  }

  function handleSearch() {
    dispatch({
      type: 'log/searchLogs',
      payload: {
        logInstanceId,
        logFileId,
        startTime: timeRange[0],
        endTime: timeRange[1],
        search: searchStr,
        logLevel,
      },
    });
  }

  function handleLoadMore() {
    dispatch({
      type: 'log/loadMoreLogs',
      payload: {
        logInstanceId,
        logFileId,
      },
    });
  }

  function handleLocalFileUploaded(res: ILogFile) {
    dispatch({ type: 'log/saveLogFile', payload: res });
    setLogFileId(res.uuid);
  }

  function disableSearch(): boolean {
    if (curUser.role === 'admin') {
      return logInstanceId === undefined || timeRange[0] === '';
    }
    if (curUser.role === 'dba') {
      return logFileId === undefined;
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
            disabled={disableSearch()}
          >
            上传
          </Button>
        )}
        {curUser.role === 'admin' && (
          <Button type="primary" disabled={disableSearch()}>
            <a
              href={`/api/v1/loginstances/${logInstanceId}.tar.gz?begin=${encodeURIComponent(
                timeRange[0],
              )}&end=${encodeURIComponent(timeRange[1])}`}
            >
              下载
            </a>
          </Button>
        )}
        {curUser.role === 'dba' && (
          <Button type="primary" onClick={() => setUploadLocalModalVisible(true)}>
            + 导入本地日志
          </Button>
        )}
      </div>
      <div className={styles.list_header}>
        {curUser.role === 'dba' && (
          <Select
            value={logFileId}
            loading={loadingLogFiles}
            allowClear
            placeholder="选择历史 log 文件"
            style={{ width: 200, marginRight: 12 }}
            onChange={(val: any) => setLogFileId(val)}
          >
            {log.logFiles.map(item => (
              <Option value={item.uuid} key={item.uuid}>
                {item.instance_name}
              </Option>
            ))}
          </Select>
        )}
        {curUser.role === 'admin' && (
          <React.Fragment>
            <Select
              value={logInstanceId}
              loading={loadingLogInstances}
              allowClear
              placeholder="选择集群实例"
              style={{ width: 200, marginRight: 12 }}
              onChange={(val: any) => setLogInstanceId(val)}
            >
              {log.logInstances.map(item => (
                <Option value={item.uuid} key={item.uuid}>
                  {item.instance_name}
                </Option>
              ))}
            </Select>
            <DatePicker.RangePicker
              style={{ marginRight: 12 }}
              showTime={{ format: 'HH:mm' }}
              format="YYYY-MM-DD HH:mm"
              placeholder={['起始时间', '结束时间']}
              onChange={handleRangeChange}
            />
          </React.Fragment>
        )}
        <Select
          allowClear
          placeholder="选择日志级别"
          style={{ width: 140, marginRight: 12 }}
          onChange={handleLogLevelChange}
        >
          {logLevels.map(item => (
            <Option value={item} key={item}>
              {item}
            </Option>
          ))}
        </Select>
        <Input
          allowClear
          onChange={(e: any) => setSearchStr(e.target.value)}
          placeholder="search"
          style={{ width: 200, height: 32, marginRight: 12 }}
        />
        <Button type="primary" disabled={disableSearch()} onClick={handleSearch}>
          搜索
        </Button>
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
        uploadUrl={`/loginstances/${logInstanceId}?begin=${encodeURIComponent(
          timeRange[0],
        )}&end=${encodeURIComponent(timeRange[1])}`}
      />
      <UploadLocalReportModal
        title="上传本地日志"
        visible={uploadLocalModalVisible}
        onClose={() => setUploadLocalModalVisible(false)}
        actionUrl="/api/v1/logfiles"
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
  loadingLogFiles: loading.effects['log/fetchLogFiles'],
}))(ReportList);
