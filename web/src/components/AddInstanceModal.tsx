import React from 'react';
import { Modal, Icon, Upload, message } from 'antd';
import { UploadProps, UploadChangeParam } from 'antd/lib/upload';
import { IInstance } from '@/models/inspection';

const { Dragger } = Upload;

interface Props {
  visible: boolean;
  onClose: () => void;
  onData: (instance: IInstance) => void;
}

function AddInstanceModal({ visible, onClose, onData }: Props) {
  const uploadProps: UploadProps = {
    name: 'file',
    accept: '.ini',
    action: '/api/v1/instances',
    showUploadList: true,
    onChange(info: UploadChangeParam) {
      const { status } = info.file;
      if (status === 'done') {
        message.success(`${info.file.name} 上传成功！`);
        onData(info.file.response as IInstance);
        onClose();
      } else if (status === 'error') {
        message.error(`${info.file.name} 上传失败，错误：${info.file.response.message}`);
      }
    },
  };

  return (
    <Modal title="添加实例" visible={visible} onCancel={onClose} okButtonProps={{ hidden: true }}>
      <Dragger {...uploadProps}>
        <p className="ant-upload-drag-icon">
          <Icon type="inbox" />
        </p>
        <p className="ant-upload-text">点击或将文件拖拽到这里上传</p>
        <p className="ant-upload-hint">仅支持目标集群创建时的 tidb-ansible/inventory.ini</p>
      </Dragger>
    </Modal>
  );
}

export default AddInstanceModal;
