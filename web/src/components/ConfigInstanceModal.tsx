import React, { useEffect, useState } from 'react';
import { Modal, Form, Checkbox, Divider, Select, message, Spin } from 'antd';
import moment from 'moment';
import { queryInstanceConfig, updateInstanceConfig } from '@/services/inspection';
import { IInstanceConfig } from '@/models/inspection';

const { Option } = Select;

const styles = require('./ConfigInstanceModal.less');

interface Props {
  visible: boolean;
  onClose: () => void;

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

function ConfigInstanceModal({ visible, onClose, instanceId }: Props) {
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [config, setConfig] = useState<IInstanceConfig | null>(null);

  useEffect(() => {
    async function fetchConfig() {
      if (instanceId === '') {
        return;
      }
      setLoading(true);
      try {
        const res = await queryInstanceConfig(instanceId);
        setConfig(res as IInstanceConfig);
      } catch (error) {
        // TODO
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

  function handleLogDurationChange(duration: number) {
    setConfig({
      ...(config as IInstanceConfig),
      collect_log_duration: duration,
    });
  }

  function handleMetricDurationChange(duration: number) {
    setConfig({
      ...(config as IInstanceConfig),
      collect_metric_duration: duration,
    });
  }

  function handleStartTimeChange(startTime: string) {
    setConfig({
      ...(config as IInstanceConfig),
      auto_sched_start: startTime,
    });
  }

  async function submit() {
    setSubmitting(true);
    try {
      await updateInstanceConfig(instanceId, config as IInstanceConfig);
      onClose();
      message.success(`${instanceId} 配置修改成功！`);
    } catch (err) {
      // TODO
    }
    setSubmitting(false);
  }

  return (
    <Modal
      visible={visible}
      onCancel={onClose}
      onOk={submit}
      title="设置"
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
              checked={config.collect_hardware_info}
            >
              硬件信息
            </Checkbox>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox
              onChange={handleChange}
              name="collect_software_info"
              checked={config.collect_software_info}
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
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <Checkbox onChange={handleChange} name="collect_log" checked={config.collect_log}>
                应用日志信息
              </Checkbox>
              <Select
                style={{ width: 120 }}
                onChange={handleLogDurationChange}
                value={config.collect_log_duration}
              >
                <Option value={60}>1 小时</Option>
                <Option value={120}>2 小时</Option>
                <Option value={240}>4 小时</Option>
              </Select>
            </div>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <Checkbox checked disabled>
                性能监控信息
              </Checkbox>
              <Select
                style={{ width: 120 }}
                onChange={handleMetricDurationChange}
                value={config.collect_metric_duration}
              >
                <Option value={60}>1 小时</Option>
                <Option value={120}>2 小时</Option>
                <Option value={240}>4 小时</Option>
              </Select>
            </div>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox checked disabled>
              告警信息
            </Checkbox>
          </Form.Item>
          <Form.Item {...formItemLayoutWithOutLabel}>
            <Checkbox onChange={handleChange} name="collect_demsg" checked={config.collect_demsg}>
              机器 demsg 信息
            </Checkbox>
          </Form.Item>
          <Divider />
          <Form.Item label="开始时间">
            <Select
              style={{ width: 120 }}
              onChange={handleStartTimeChange}
              value={config.auto_sched_start}
            >
              {oneDayTimes.map(time => (
                <Option value={time} key={time}>
                  {time}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Divider />
          <Form.Item label="报告收集频率">
            <span>每日 1 次</span>
          </Form.Item>
          <Form.Item label="报告保存时长">
            <span>30 天</span>
          </Form.Item>
        </Form>
      )}
    </Modal>
  );
}

export default ConfigInstanceModal;
