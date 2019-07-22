import React, { useState } from 'react';
import { Modal, Input, message } from 'antd';

const styles = require('./AddMiscReportModal.less');

interface Props {
  visible: boolean;
  onClose: () => void;

  onData: (machine: string) => Promise<any>;
}

function AddMiscReportModal({ visible, onClose, onData }: Props) {
  const [submitting, setSubmitting] = useState(false);
  const [machine, setMachine] = useState('');

  async function submit() {
    setSubmitting(true);
    try {
      await onData(machine);
      onClose();
      message.success(`正在收集 ${machine} 的新报告！`);
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
      title="获取报告"
      confirmLoading={submitting}
      okButtonProps={{ disabled: machine.length === 0 }}
    >
      <div className={styles.modal_item}>
        <span>IP:端口</span>
        <Input value={machine} onChange={(e: any) => setMachine(e.target.value)} />
      </div>
    </Modal>
  );
}

export default AddMiscReportModal;
