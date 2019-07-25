import React, { useState } from 'react';
import { Modal, message } from 'antd';
import request from '@/utils/request';

interface Props {
  visible: boolean;
  onClose: () => void;

  uploadUrl: string;
}

function UploadRemoteReportModal({ visible, onClose, uploadUrl }: Props) {
  const [submitting, setSubmitting] = useState(false);

  async function submit() {
    setSubmitting(true);
    try {
      await request.put(uploadUrl);
      onClose();
      message.success('报告已上传');
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
      okText="上传"
      title="上传报告"
      confirmLoading={submitting}
    >
      <p>确定要上传这个报告吗？</p>
    </Modal>
  );
}

export default UploadRemoteReportModal;
