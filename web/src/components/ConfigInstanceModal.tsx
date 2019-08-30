import React, { useEffect, useState } from 'react';
import { Modal, Form, Checkbox, Divider, Select, message, Spin, DatePicker, Row, Col } from 'antd';
import moment from 'moment';
import { RangePickerValue } from 'antd/lib/date-picker/interface';
import { CheckboxValueType } from 'antd/lib/checkbox/Group';
import { formatMessage } from 'umi-plugin-react/locale';
import { queryInstanceConfig, updateInstanceConfig } from '@/services/inspection';
import { IInstanceConfig } from '@/models/inspection';
import { Dispatch } from '@/models/connect';

const { Option } = Select;

const styles = require('./ConfigInstanceModal.less');

interface Props {
  visible: boolean;
  onClose: () => void;

  dispatch?: Dispatch;

  manual: boolean;

  instanceId: string;
}

const formItemLayout = {
  labelCol: {
    xs: { span: 24 },
    sm: { span: 8 },
  },
  wrapperCol: {
    xs: { span: 24 },
    sm: { span: 16 },
  },
};
const formItemLayoutWithOutLabel = {
  wrapperCol: {
    xs: { span: 24, offset: 0 },
    sm: { span: 16, offset: 8 },
  },
};

const oneDayTimes: string[] = Array(48)
  .fill(0)
  .map((_, index) =>
    moment([2019])
      .minutes(index * 30)
      .format('HH:mm'),
  );

const endOfToday = moment().endOf('day');

const weekDays = 'MON,TUE,WED,THU,FRI,SAT,SUN'.split(',');

function ConfigInstanceModal({ visible, onClose, dispatch, manual, instanceId }: Props) {
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [config, setConfig] = useState<IInstanceConfig | null>(null);

  const defTimeRange: [moment.Moment, moment.Moment] = [moment().subtract(1, 'hours'), moment()];

  useEffect(() => {
    async function fetchConfig() {
      if (instanceId === '') {
        return;
      }
      setLoading(true);
      const res = await queryInstanceConfig(instanceId);
      if (res !== undefined) {
        setConfig({
          ...(res as IInstanceConfig),
          manual_sched_range: [defTimeRange[0].format(), defTimeRange[1].format()],
        });
      }
      setLoading(false);
    }
    fetchConfig();
  }, [instanceId]);

  function handleChange(event: any) {
    const { target } = event;
    const value = target.type === 'checkbox' ? target.checked : target.value;
    setConfig({
      ...(config as IInstanceConfig),
      [target.name]: value,
    });
  }

  function handleAutoSchedStartTimeChange(startTime: string) {
    setConfig({
      ...(config as IInstanceConfig),
      auto_sched_start: startTime,
    });
  }

  function handleAutoSchedDurationChange(duration: number) {
    setConfig({
      ...(config as IInstanceConfig),
      auto_sched_duration: duration,
    });
  }

  function handleManualSchedRangeChange(dates: RangePickerValue, dateStrings: [string, string]) {
    // 如果用户进行了 clear，dates 为 [], dateStrings 为 ["", ""]
    let timeRange: [string, string] = ['', ''];
    if (dates[0] && dates[1]) {
      timeRange = [dates[0].format(), dates[1].format()];
    }
    setConfig({
      ...(config as IInstanceConfig),
      manual_sched_range: timeRange,
    });
  }

  function disableDate(current: moment.Moment | undefined) {
    // Can not select days before today
    return (current && current > endOfToday) || false;
  }

  function handleWeekdaysChange(checkedValues: CheckboxValueType[]) {
    setConfig({
      ...(config as IInstanceConfig),
      auto_sched_day: checkedValues.join(','),
    });
  }

  async function submit() {
    setSubmitting(true);

    if (manual && dispatch) {
      dispatch({
        type: 'inspection/addInspection',
        payload: {
          instanceId,
          config,
        },
      }).then((ret: boolean) => {
        setSubmitting(false);
        if (ret) {
          onClose();
          message.success(`${instanceId} 手动诊断成功！`);
        }
      });
    } else if (!manual) {
      const res = await updateInstanceConfig(instanceId, config as IInstanceConfig);
      setSubmitting(false);
      if (res !== undefined) {
        onClose();
        message.success(`${instanceId} 自动诊断配置修改成功！`);
      }
    }
  }

  return (
    <Modal
      width={manual ? 600 : 520}
      visible={visible}
      onCancel={onClose}
      onOk={submit}
      title={manual ? '手动诊断设置' : '自动诊断设置'}
      confirmLoading={submitting}
      okButtonProps={{ disabled: loading }}
    >
      {loading && <Spin size="small" style={{ marginLeft: 8, marginRight: 8 }} />}
      {!loading && config && (
        <Form labelAlign="left" {...formItemLayout} className={styles.config_form}>
          <Form.Item label="信息收集项">
            <Checkbox
              onChange={handleChange}
              name="collect_hardware_info"
              // checked={config.collect_hardware_info}
              checked
              disabled
            >
              硬件信息
            </Checkbox>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox
              onChange={handleChange}
              name="collect_software_info"
              // checked={config.collect_software_info}
              checked
              disabled
            >
              软件信息
            </Checkbox>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox checked disabled>
              配置信息
            </Checkbox>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox checked disabled>
              网络质量信息
            </Checkbox>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox checked disabled>
              NTP 同步信息
            </Checkbox>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox checked disabled>
              慢查询
            </Checkbox>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox
              onChange={handleChange}
              name="collect_log"
              // checked={config.collect_log}
              checked
              disabled
            >
              应用日志信息
            </Checkbox>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox checked disabled>
              性能监控信息
            </Checkbox>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox checked disabled>
              告警信息
            </Checkbox>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox
              onChange={handleChange}
              name="collect_demsg"
              //  checked={config.collect_demsg}
              checked
              disabled
            >
              机器 demsg 信息
            </Checkbox>
          </Form.Item>
          <Divider />
          {!manual && (
            <React.Fragment>
              <Form.Item label="开始时间">
                <Select
                  style={{ width: 120 }}
                  onChange={handleAutoSchedStartTimeChange}
                  value={config.auto_sched_start}
                >
                  {oneDayTimes.map(time => (
                    <Option value={time} key={time}>
                      {time}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item label="统计信息时长">
                <Select
                  style={{ width: 120 }}
                  onChange={handleAutoSchedDurationChange}
                  value={config.auto_sched_duration}
                >
                  <Option value={60}>1 小时</Option>
                  <Option value={60 * 2}>2 小时</Option>
                  <Option value={60 * 4}>4 小时</Option>
                </Select>
              </Form.Item>
              <Form.Item label="报告收集频率">
                <Checkbox.Group
                  style={{ paddingTop: 10 }}
                  defaultValue={config.auto_sched_day.split(',')}
                  onChange={handleWeekdaysChange}
                >
                  <Row>
                    {weekDays.map(day => (
                      <Col span={8} key={day}>
                        <Checkbox value={day}>{formatMessage({ id: `days.${day}` })}</Checkbox>
                      </Col>
                    ))}
                  </Row>
                </Checkbox.Group>
              </Form.Item>
              <Divider />
            </React.Fragment>
          )}
          {manual && (
            <React.Fragment>
              <Form.Item label="统计信息时间段">
                <DatePicker.RangePicker
                  showTime={{ format: 'HH:mm' }}
                  format="YYYY-MM-DD HH:mm"
                  placeholder={['起始时间', '结束时间']}
                  defaultValue={defTimeRange}
                  disabledDate={disableDate}
                  onChange={handleManualSchedRangeChange}
                />
              </Form.Item>
              <Divider />
            </React.Fragment>
          )}
          <Form.Item label="报告保存时长">
            <span>30 天</span>
          </Form.Item>
        </Form>
      )}
    </Modal>
  );
}

export default ConfigInstanceModal;
