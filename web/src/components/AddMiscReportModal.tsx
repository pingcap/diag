import React, { useState } from 'react';
import { Modal, message, Select } from 'antd';
import { IFormatInstance } from '@/models/inspection';

const { Option } = Select;
const styles = require('./AddMiscReportModal.less');

interface Props {
  visible: boolean;
  onClose: () => void;

  instances: IFormatInstance[];

  onData: (instanceId: string) => Promise<any>;
}

function AddMiscReportModal({ visible, onClose, instances, onData }: Props) {
  const [submitting, setSubmitting] = useState(false);
  const [instanceId, setInstanceId] = useState('');

  async function submit() {
    setSubmitting(true);
    try {
      await onData(instanceId);
      onClose();
      message.success(`正在收集 ${instanceId} 的新报告！`);
    } catch (err) {
      // TODO
    }
    setSubmitting(false);
  }

  function handleSelectChange(value: string | undefined) {
    // 如果用户对 select 进行了 clear，value 为 undefined
    setInstanceId(value || '');
  }

  return (
    <Modal
      visible={visible}
      onCancel={onClose}
      onOk={submit}
      title="获取报告"
      confirmLoading={submitting}
      okButtonProps={{ disabled: instanceId === '' }}
    >
      <div className={styles.modal_item}>
        <span style={{ marginRight: 20 }}>选择集群实例</span>
        <Select style={{ width: 200 }} onChange={handleSelectChange}>
          {instances.map(item => (
            <Option value={item.uuid} key={item.uuid}>
              {item.name}
            </Option>
          ))}
        </Select>
      </div>
    </Modal>
  );
}

export default AddMiscReportModal;
