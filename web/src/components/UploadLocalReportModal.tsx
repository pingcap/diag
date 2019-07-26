import React from 'react';
import { Modal, Icon, Upload, message } from 'antd';
import { UploadProps, UploadChangeParam } from 'antd/lib/upload';

const { Dragger } = Upload;

interface Props {
  visible: boolean;
  actionUrl: string;
  title?: string;
  onClose: () => void;
  onData: (data: any) => void;
}

function UploadLocalReportModal({ visible, actionUrl, title, onClose, onData }: Props) {
  const uploadProps: UploadProps = {
    name: 'file',
    accept: '.gz',
    action: actionUrl,
    showUploadList: true,
    onChange(info: UploadChangeParam) {
      const { status } = info.file;
      if (status === 'done') {
        message.success(`${info.file.name} 上传成功！`);
        onData(info.file.response);
        onClose();
      } else if (status === 'error') {
        message.error(`${info.file.name} 上传失败，错误：${info.file.response.error}`);
      }
    },
  };

  return (
    <Modal
      title={title || '上传本地报告'}
      visible={visible}
      onCancel={onClose}
      okButtonProps={{ hidden: true }}
    >
      <Dragger {...uploadProps}>
        <p className="ant-upload-drag-icon">
          <Icon type="inbox" />
        </p>
        <p className="ant-upload-text">点击或将文件拖拽到这里上传</p>
        <p className="ant-upload-hint">仅支持 .tar.gz 格式的压缩包</p>
      </Dragger>
    </Modal>
  );
}

export default UploadLocalReportModal;
