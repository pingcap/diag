import React, { useEffect, useState } from 'react';
import { Table, Button, DatePicker, Input, Select } from 'antd';
import { connect } from 'dva';
import { RangePickerValue } from 'antd/lib/date-picker/interface';
import { ConnectState, ConnectProps, Dispatch } from '@/models/connect';
import { LogModelState } from '@/models/log';

const { Option } = Select;

const styles = require('../style.less');

const tableColumns = [
  {
    title: '时间',
    dataIndex: 'time',
    key: 'time',
  },
  {
    title: '文件名',
    dataIndex: 'file',
    key: 'file',
  },
  {
    title: '内容',
    dataIndex: 'content',
    key: 'content',
  },
];

interface ReportListProps extends ConnectProps {
  dispatch: Dispatch;

  log: LogModelState;
  searchingLogs: boolean;
  loadingMoreLogs: boolean;
  loadingLogInstances: boolean;
}

function ReportList({
  dispatch,
  log,
  searchingLogs,
  loadingMoreLogs,
  loadingLogInstances,
}: ReportListProps) {
  const [selectedInstanceId, setSelectedInstanceId] = useState('');
  const [timeRange, setTimeRange] = useState<[string, string]>(['', '']);

  useEffect(() => {
    dispatch({ type: 'log/fetchLogInstances' });
    dispatch({ type: 'log/resetLogs' });
  }, []);

  function handleSelectChange(value: string | undefined) {
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
        startTime: timeRange[0],
        endTime: timeRange[1],
        search: value,
      },
    });
  }

  function handleDownlaod() {}

  function handleLoadMore() {
    dispatch({
      type: 'log/loadMoreLogs',
      payload: selectedInstanceId,
    });
  }

  return (
    <div className={styles.container}>
      <div className={styles.list_header}>
        <h2>日志搜索</h2>
        <Select
          loading={loadingLogInstances}
          allowClear
          placeholder="选择集群实例"
          style={{ width: 140, marginLeft: 12 }}
          onChange={handleSelectChange}
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
        <Input.Search
          allowClear
          disabled={selectedInstanceId.length === 0 || timeRange[0].length === 0}
          placeholder="search"
          onSearch={handleSearch}
          style={{ width: 200, height: 32 }}
          size="small"
        />
        <div className={styles.space} />
        <Button type="primary" onClick={handleDownlaod}>
          下载
        </Button>
      </div>
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
    </div>
  );
}

export default connect(({ log, loading }: ConnectState) => ({
  log,
  searchingLogs: loading.effects['log/searchLogs'],
  loadingMoreLogs: loading.effects['log/loadMoreLogs'],
  loadingLogInstances: loading.effects['log/fetchLogInstances'],
}))(ReportList);
